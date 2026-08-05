//! Platform-neutral policy engine for events collected by an eBPF socket probe.

use std::collections::HashSet;
use std::net::IpAddr;
use std::str::FromStr;
use std::sync::{Arc, Mutex};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Protocol { Http, Tcp }

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OutboundEvent { pub process: String, pub destination: IpAddr, pub port: u16, pub protocol: Protocol }

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Violation { pub event: OutboundEvent, pub reason: String }

#[derive(Clone, Default)]
pub struct NetworkPolicy { allowed_hosts: HashSet<IpAddr>, allowed_ports: HashSet<u16>, allowed_processes: HashSet<String> }

impl NetworkPolicy {
    pub fn allow_host(mut self, host: &str) -> Result<Self, std::net::AddrParseError> { self.allowed_hosts.insert(IpAddr::from_str(host)?); Ok(self) }
    pub fn allow_port(mut self, port: u16) -> Self { self.allowed_ports.insert(port); self }
    pub fn allow_process(mut self, process: impl Into<String>) -> Self { self.allowed_processes.insert(process.into()); self }
}

#[derive(Clone, Default)]
pub struct NetworkTracer { policy: Arc<NetworkPolicy>, violations: Arc<Mutex<Vec<Violation>>> }

impl NetworkTracer {
    pub fn new(policy: NetworkPolicy) -> Self { Self { policy: Arc::new(policy), violations: Arc::new(Mutex::new(Vec::new())) } }
    /// Consumes an event emitted by an eBPF socket/connect probe and reports policy violations.
    pub fn observe(&self, event: OutboundEvent) -> Option<Violation> {
        let reason = if !self.policy.allowed_processes.is_empty() && !self.policy.allowed_processes.contains(&event.process) { Some("process is not approved") }
        else if !self.policy.allowed_hosts.is_empty() && !self.policy.allowed_hosts.contains(&event.destination) { Some("destination host is not allowlisted") }
        else if !self.policy.allowed_ports.is_empty() && !self.policy.allowed_ports.contains(&event.port) { Some("destination port is not allowlisted") }
        else { None };
        reason.map(|reason| { let violation = Violation { event, reason: reason.to_owned() }; self.violations.lock().expect("violation lock poisoned").push(violation.clone()); violation })
    }
    pub fn violations(&self) -> Vec<Violation> { self.violations.lock().expect("violation lock poisoned").clone() }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn event(process: &str, destination: &str, port: u16) -> OutboundEvent { OutboundEvent { process: process.into(), destination: destination.parse().unwrap(), port, protocol: Protocol::Http } }
    #[test] fn permits_approved_traffic() { let policy = NetworkPolicy::default().allow_process("mcp-server").allow_host("203.0.113.10").unwrap().allow_port(443); assert_eq!(NetworkTracer::new(policy).observe(event("mcp-server", "203.0.113.10", 443)), None); }
    #[test] fn records_destination_and_process_violations() { let policy = NetworkPolicy::default().allow_process("mcp-server").allow_host("203.0.113.10").unwrap().allow_port(443); let tracer = NetworkTracer::new(policy); let violation = tracer.observe(event("agent", "198.51.100.2", 80)).unwrap(); assert_eq!(violation.reason, "process is not approved"); assert_eq!(tracer.violations().len(), 1); }
    #[test] fn detects_unapproved_port() { let policy = NetworkPolicy::default().allow_port(443); let tracer = NetworkTracer::new(policy); assert_eq!(tracer.observe(event("mcp", "127.0.0.1", 80)).unwrap().reason, "destination port is not allowlisted"); }
}
