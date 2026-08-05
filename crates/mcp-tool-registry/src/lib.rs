//! Thread-safe, in-memory discovery registry for MCP tools.

use std::collections::HashMap;
use std::sync::{Arc, RwLock};

/// Metadata advertised by a tool implementation.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Tool {
    pub name: String,
    pub description: String,
    pub input_schema: String,
    pub tags: Vec<String>,
}

impl Tool {
    pub fn new(
        name: impl Into<String>,
        description: impl Into<String>,
        input_schema: impl Into<String>,
        tags: impl IntoIterator<Item = impl Into<String>>,
    ) -> Self {
        Self {
            name: name.into(),
            description: description.into(),
            input_schema: input_schema.into(),
            tags: tags.into_iter().map(Into::into).collect(),
        }
    }
}

/// Errors returned by registry mutations and lookups.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum RegistryError {
    InvalidName,
    AlreadyExists(String),
    NotFound(String),
    Poisoned,
}

/// A cloneable registry whose readers can discover tools concurrently.
#[derive(Clone, Default)]
pub struct ToolRegistry {
    tools: Arc<RwLock<HashMap<String, Tool>>>,
}

impl ToolRegistry {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn register(&self, tool: Tool) -> Result<(), RegistryError> {
        validate_name(&tool.name)?;
        let mut tools = self.tools.write().map_err(|_| RegistryError::Poisoned)?;
        if tools.contains_key(&tool.name) {
            return Err(RegistryError::AlreadyExists(tool.name));
        }
        tools.insert(tool.name.clone(), tool);
        Ok(())
    }

    /// Replaces a registered tool atomically. The tool must already exist.
    pub fn update(&self, tool: Tool) -> Result<(), RegistryError> {
        validate_name(&tool.name)?;
        let mut tools = self.tools.write().map_err(|_| RegistryError::Poisoned)?;
        if !tools.contains_key(&tool.name) {
            return Err(RegistryError::NotFound(tool.name));
        }
        tools.insert(tool.name.clone(), tool);
        Ok(())
    }

    pub fn get(&self, name: &str) -> Result<Option<Tool>, RegistryError> {
        Ok(self
            .tools
            .read()
            .map_err(|_| RegistryError::Poisoned)?
            .get(name)
            .cloned())
    }

    /// Returns tools matching every requested tag, sorted by tool name.
    pub fn discover(&self, tags: &[&str]) -> Result<Vec<Tool>, RegistryError> {
        let mut result: Vec<_> = self
            .tools
            .read()
            .map_err(|_| RegistryError::Poisoned)?
            .values()
            .filter(|tool| tags.iter().all(|tag| tool.tags.iter().any(|owned| owned == tag)))
            .cloned()
            .collect();
        result.sort_by(|left, right| left.name.cmp(&right.name));
        Ok(result)
    }

    pub fn unregister(&self, name: &str) -> Result<Tool, RegistryError> {
        self.tools
            .write()
            .map_err(|_| RegistryError::Poisoned)?
            .remove(name)
            .ok_or_else(|| RegistryError::NotFound(name.to_owned()))
    }

    pub fn len(&self) -> Result<usize, RegistryError> {
        Ok(self.tools.read().map_err(|_| RegistryError::Poisoned)?.len())
    }

    pub fn is_empty(&self) -> Result<bool, RegistryError> {
        Ok(self.len()? == 0)
    }
}

fn validate_name(name: &str) -> Result<(), RegistryError> {
    if name.is_empty()
        || !name
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b'.'))
    {
        return Err(RegistryError::InvalidName);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread;

    fn tool(name: &str, tags: &[&str]) -> Tool {
        Tool::new(name, "description", "{}", tags.iter().copied())
    }

    #[test]
    fn registers_discovers_updates_and_removes_tools() {
        let registry = ToolRegistry::new();
        registry.register(tool("repo.search", &["git", "read"])).unwrap();
        registry.register(tool("repo.write", &["git", "write"])).unwrap();

        let discovered = registry.discover(&["git", "read"]).unwrap();
        assert_eq!(discovered.len(), 1);
        assert_eq!(discovered[0].name, "repo.search");

        registry.update(tool("repo.search", &["git", "indexed"])).unwrap();
        assert!(registry.discover(&["read"]).unwrap().is_empty());
        assert_eq!(registry.unregister("repo.search").unwrap().name, "repo.search");
        assert_eq!(registry.len().unwrap(), 1);
    }

    #[test]
    fn rejects_duplicate_invalid_and_missing_tools() {
        let registry = ToolRegistry::new();
        assert_eq!(registry.register(tool("", &[])), Err(RegistryError::InvalidName));
        registry.register(tool("safe", &[])).unwrap();
        assert_eq!(registry.register(tool("safe", &[])), Err(RegistryError::AlreadyExists("safe".into())));
        assert_eq!(registry.unregister("missing"), Err(RegistryError::NotFound("missing".into())));
    }

    #[test]
    fn supports_concurrent_registration() {
        let registry = ToolRegistry::new();
        let handles: Vec<_> = (0..24)
            .map(|index| {
                let registry = registry.clone();
                thread::spawn(move || registry.register(tool(&format!("tool-{index}"), &["parallel"])))
            })
            .collect();
        for handle in handles {
            handle.join().unwrap().unwrap();
        }
        assert_eq!(registry.discover(&["parallel"]).unwrap().len(), 24);
    }
}
