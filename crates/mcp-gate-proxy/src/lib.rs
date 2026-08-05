//! Policy-enforcing gateway for MCP tool invocations.

use std::collections::{HashMap, HashSet};
use std::sync::{Arc, Mutex, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Invocation {
    pub agent_id: String,
    pub tool: String,
    pub payload: String,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Decision {
    Allowed,
    DeniedTool,
    DeniedPii,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AuditEntry {
    pub timestamp_ms: u128,
    pub agent_id: String,
    pub tool: String,
    pub payload_fingerprint: u64,
    pub decision: Decision,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ProxyResponse {
    pub decision: Decision,
    pub message: String,
}

#[derive(Default)]
pub struct GateProxy {
    allowlist: Arc<RwLock<HashMap<String, HashSet<String>>>>,
    audit: Arc<Mutex<Vec<AuditEntry>>>,
}

impl GateProxy {
    pub fn new() -> Self { Self::default() }

    pub fn allow_tool(&self, agent_id: impl Into<String>, tool: impl Into<String>) {
        let mut allowlist = self.allowlist.write().expect("allowlist lock poisoned");
        allowlist.entry(agent_id.into()).or_default().insert(tool.into());
    }

    pub fn revoke_tool(&self, agent_id: &str, tool: &str) {
        if let Some(tools) = self.allowlist.write().expect("allowlist lock poisoned").get_mut(agent_id) {
            tools.remove(tool);
        }
    }

    /// Validates and records an invocation without performing the external side effect.
    pub async fn inspect(&self, invocation: Invocation) -> ProxyResponse {
        let decision = if !self.is_allowed(&invocation.agent_id, &invocation.tool) {
            Decision::DeniedTool
        } else if contains_pii(&invocation.payload) {
            Decision::DeniedPii
        } else {
            Decision::Allowed
        };
        self.append_audit(&invocation, decision.clone());
        let message = match decision {
            Decision::Allowed => "invocation authorized".to_owned(),
            Decision::DeniedTool => "tool is not allowlisted for this agent".to_owned(),
            Decision::DeniedPii => "payload contains protected personal data".to_owned(),
        };
        ProxyResponse { decision, message }
    }

    pub fn audit_log(&self) -> Vec<AuditEntry> {
        self.audit.lock().expect("audit lock poisoned").clone()
    }

    fn is_allowed(&self, agent_id: &str, tool: &str) -> bool {
        self.allowlist.read().expect("allowlist lock poisoned").get(agent_id).is_some_and(|tools| tools.contains(tool))
    }

    fn append_audit(&self, invocation: &Invocation, decision: Decision) {
        let timestamp_ms = SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default().as_millis();
        self.audit.lock().expect("audit lock poisoned").push(AuditEntry {
            timestamp_ms,
            agent_id: invocation.agent_id.clone(),
            tool: invocation.tool.clone(),
            payload_fingerprint: fingerprint(&invocation.payload),
            decision,
        });
    }
}

fn contains_pii(value: &str) -> bool {
    value.split(|character: char| character.is_whitespace() || matches!(character, ',' | ';' | ':'))
        .any(|token| is_email(token) || is_tckn(token) || is_card(token))
}

fn is_email(token: &str) -> bool {
    token.split_once('@').is_some_and(|(local, domain)| !local.is_empty() && domain.contains('.') && !domain.ends_with('.'))
}

fn is_tckn(token: &str) -> bool {
    if token.len() != 11 || token.starts_with('0') || !token.bytes().all(|byte| byte.is_ascii_digit()) { return false; }
    let digits: Vec<i32> = token.bytes().map(|byte| i32::from(byte - b'0')).collect();
    let tenth = ((digits[0] + digits[2] + digits[4] + digits[6] + digits[8]) * 7 - (digits[1] + digits[3] + digits[5] + digits[7])).rem_euclid(10);
    digits[9] == tenth && digits[..10].iter().sum::<i32>() % 10 == digits[10]
}

fn is_card(token: &str) -> bool {
    if !(13..=19).contains(&token.len()) || !token.bytes().all(|byte| byte.is_ascii_digit()) { return false; }
    token.bytes().rev().enumerate().map(|(index, byte)| {
        let mut digit = u32::from(byte - b'0');
        if index % 2 == 1 { digit = if digit > 4 { digit * 2 - 9 } else { digit * 2 }; }
        digit
    }).sum::<u32>() % 10 == 0
}

fn fingerprint(value: &str) -> u64 {
    value.bytes().fold(0xcbf29ce484222325, |hash, byte| (hash ^ u64::from(byte)).wrapping_mul(0x100000001b3))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::future::Future;
    use std::pin::Pin;
    use std::task::{Context, Poll, RawWaker, RawWakerVTable, Waker};

    fn block_on<F: Future>(future: F) -> F::Output {
        fn noop(_: *const ()) {}
        fn clone(_: *const ()) -> RawWaker { RawWaker::new(std::ptr::null(), &RawWakerVTable::new(clone, noop, noop, noop)) }
        let raw = RawWaker::new(std::ptr::null(), &RawWakerVTable::new(clone, noop, noop, noop));
        let waker = unsafe { Waker::from_raw(raw) };
        let mut context = Context::from_waker(&waker);
        let mut future = std::pin::pin!(future);
        match Pin::as_mut(&mut future).poll(&mut context) { Poll::Ready(value) => value, Poll::Pending => panic!("inspection must complete immediately") }
    }

    fn invocation(tool: &str, payload: &str) -> Invocation { Invocation { agent_id: "agent-a".into(), tool: tool.into(), payload: payload.into() } }

    #[test]
    fn authorizes_allowlisted_clean_invocation_and_audits_it() {
        let proxy = GateProxy::new();
        proxy.allow_tool("agent-a", "github.issue.create");
        assert_eq!(block_on(proxy.inspect(invocation("github.issue.create", "title=fix"))).decision, Decision::Allowed);
        let entries = proxy.audit_log();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].decision, Decision::Allowed);
        assert_ne!(entries[0].payload_fingerprint, 0);
    }

    #[test]
    fn rejects_non_allowlisted_tool_and_pii_payload() {
        let proxy = GateProxy::new();
        assert_eq!(block_on(proxy.inspect(invocation("db.drop", "safe"))).decision, Decision::DeniedTool);
        proxy.allow_tool("agent-a", "db.query");
        assert_eq!(block_on(proxy.inspect(invocation("db.query", "email=ada@example.com"))).decision, Decision::DeniedPii);
        assert_eq!(proxy.audit_log().len(), 2);
    }
}
