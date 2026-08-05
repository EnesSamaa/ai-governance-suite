//! Fixed-capacity, lock-free telemetry buffer for non-sensitive event metadata.

use std::sync::atomic::{AtomicU64, AtomicU8, AtomicUsize, Ordering};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
#[repr(u8)]
pub enum EventKind { AgentAction = 1, CommandInput = 2, ToolCall = 3 }

impl EventKind { fn from_byte(value: u8) -> Option<Self> { match value { 1 => Some(Self::AgentAction), 2 => Some(Self::CommandInput), 3 => Some(Self::ToolCall), _ => None } } }

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct TelemetryEvent { pub sequence: u64, pub timestamp_ns: u64, pub kind: EventKind, pub subject_hash: u64 }

struct Slot { state: AtomicU64, timestamp_ns: AtomicU64, kind: AtomicU8, subject_hash: AtomicU64 }
impl Slot { fn new() -> Self { Self { state: AtomicU64::new(0), timestamp_ns: AtomicU64::new(0), kind: AtomicU8::new(0), subject_hash: AtomicU64::new(0) } } }

pub struct TelemetryRing { slots: Box<[Slot]>, cursor: AtomicUsize }

impl TelemetryRing {
    pub fn new(capacity: usize) -> Self { assert!(capacity > 0); Self { slots: (0..capacity).map(|_| Slot::new()).collect(), cursor: AtomicUsize::new(0) } }
    pub fn record(&self, timestamp_ns: u64, kind: EventKind, subject: &str) {
        let sequence = self.cursor.fetch_add(1, Ordering::Relaxed) as u64;
        let slot = &self.slots[sequence as usize % self.slots.len()];
        slot.state.store(sequence * 2 + 1, Ordering::Release);
        slot.timestamp_ns.store(timestamp_ns, Ordering::Relaxed);
        slot.kind.store(kind as u8, Ordering::Relaxed);
        slot.subject_hash.store(fingerprint(subject), Ordering::Relaxed);
        slot.state.store(sequence * 2 + 2, Ordering::Release);
    }
    pub fn snapshot(&self) -> Vec<TelemetryEvent> {
        let mut events = Vec::new();
        for slot in self.slots.iter() {
            let before = slot.state.load(Ordering::Acquire);
            if before == 0 || before % 2 == 1 { continue; }
            let event = EventKind::from_byte(slot.kind.load(Ordering::Relaxed)).map(|kind| TelemetryEvent { sequence: before / 2 - 1, timestamp_ns: slot.timestamp_ns.load(Ordering::Relaxed), kind, subject_hash: slot.subject_hash.load(Ordering::Relaxed) });
            if before == slot.state.load(Ordering::Acquire) { if let Some(event) = event { events.push(event); } }
        }
        events.sort_by_key(|event| event.sequence);
        events
    }
    pub fn capacity(&self) -> usize { self.slots.len() }
}

fn fingerprint(value: &str) -> u64 { value.bytes().fold(0xcbf29ce484222325, |hash, byte| (hash ^ u64::from(byte)).wrapping_mul(0x100000001b3)) }

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use std::thread;
    #[test] fn records_events_without_retaining_raw_subject() { let ring = TelemetryRing::new(4); ring.record(10, EventKind::ToolCall, "github.create"); let events = ring.snapshot(); assert_eq!(events[0].kind, EventKind::ToolCall); assert_ne!(events[0].subject_hash, 0); }
    #[test] fn overwrites_oldest_event_when_full() { let ring = TelemetryRing::new(2); for index in 0..3 { ring.record(index, EventKind::AgentAction, "x"); } let events = ring.snapshot(); assert_eq!(events.iter().map(|event| event.sequence).collect::<Vec<_>>(), vec![1, 2]); }
    #[test] fn accepts_concurrent_writers() { let ring = Arc::new(TelemetryRing::new(128)); let workers: Vec<_> = (0..8).map(|index| { let ring = ring.clone(); thread::spawn(move || for item in 0..16 { ring.record((index * 16 + item) as u64, EventKind::CommandInput, "cmd"); }) }).collect(); for worker in workers { worker.join().unwrap(); } assert_eq!(ring.snapshot().len(), 128); }
}
