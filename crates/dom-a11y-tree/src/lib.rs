//! Compact accessibility-tree extraction for agent-facing DOM understanding.

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct A11yNode { pub role: String, pub name: String, pub children: Vec<A11yNode> }

pub fn parse_html(html: &str) -> A11yNode {
    let mut root = A11yNode { role: "document".into(), name: String::new(), children: Vec::new() };
    let mut cursor = 0;
    while let Some(open) = html[cursor..].find('<') {
        let start = cursor + open;
        let Some(end_rel) = html[start..].find('>') else { break };
        let end = start + end_rel;
        let tag = &html[start + 1..end];
        cursor = end + 1;
        if tag.starts_with('/') || tag.starts_with('!') { continue; }
        let name = tag.split_whitespace().next().unwrap_or("").trim_end_matches('/').to_ascii_lowercase();
        if name.is_empty() || has_attr(tag, "hidden") || attr(tag, "aria-hidden").as_deref() == Some("true") { continue; }
        let role = attr(tag, "role").unwrap_or_else(|| implicit_role(&name).to_owned());
        if role.is_empty() { continue; }
        let accessible_name = attr(tag, "aria-label").or_else(|| attr(tag, "title")).unwrap_or_else(|| text_until_tag(&html[cursor..]));
        root.children.push(A11yNode { role, name: normalize(&accessible_name), children: Vec::new() });
    }
    root
}

pub fn flatten(node: &A11yNode) -> Vec<(String, String)> {
    let mut output = vec![(node.role.clone(), node.name.clone())];
    for child in &node.children { output.extend(flatten(child)); }
    output
}

fn implicit_role(tag: &str) -> &'static str { match tag { "a" => "link", "button" => "button", "input" => "textbox", "img" => "img", "nav" => "navigation", "main" => "main", "h1"|"h2"|"h3"|"h4"|"h5"|"h6" => "heading", "li" => "listitem", _ => "" } }
fn has_attr(tag: &str, name: &str) -> bool { tag.split_whitespace().any(|part| part.trim_end_matches('/').eq_ignore_ascii_case(name)) }
fn attr(tag: &str, name: &str) -> Option<String> {
    let lower = tag.to_ascii_lowercase(); let needle = format!("{name}="); let position = lower.find(&needle)?; let value = &tag[position + needle.len()..];
    let quote = value.chars().next()?; if quote == '"' || quote == '\'' { value[1..].find(quote).map(|end| value[1..1 + end].to_owned()) } else { Some(value.split_whitespace().next()?.trim_end_matches('/').to_owned()) }
}
fn text_until_tag(value: &str) -> String { value.split('<').next().unwrap_or("").to_owned() }
fn normalize(value: &str) -> String { value.split_whitespace().collect::<Vec<_>>().join(" ") }

#[cfg(test)]
mod tests { use super::*;
    #[test] fn extracts_semantic_nodes_without_hidden_content() { let tree = parse_html("<nav aria-label='Primary'><a href='/'>Home</a></nav><button> Save </button><div hidden>secret</div>"); assert_eq!(tree.children, vec![A11yNode { role: "navigation".into(), name: "Primary".into(), children: vec![] }, A11yNode { role: "link".into(), name: "Home".into(), children: vec![] }, A11yNode { role: "button".into(), name: "Save".into(), children: vec![] }]); }
    #[test] fn supports_explicit_roles_and_flattening() { let tree = parse_html("<div role='dialog' aria-label='Confirm'></div>"); assert_eq!(flatten(&tree), vec![("document".into(), "".into()), ("dialog".into(), "Confirm".into())]); }
}
