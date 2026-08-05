//! Terminal-independent interactive diff state for a Ratatui/Crossterm frontend.

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum DiffKind { Added, Removed, Context }
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DiffLine { pub kind: DiffKind, pub text: String }
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Viewport { pub lines: Vec<DiffLine>, pub selected: usize, pub offset: usize }
impl Viewport {
    pub fn from_unified(diff: &str) -> Self { let lines = diff.lines().filter(|line| !line.starts_with("+++") && !line.starts_with("---") && !line.starts_with("@@")).map(|line| { let (kind, text) = match line.as_bytes().first() { Some(b'+') => (DiffKind::Added, &line[1..]), Some(b'-') => (DiffKind::Removed, &line[1..]), _ => (DiffKind::Context, line) }; DiffLine { kind, text: text.to_owned() } }).collect(); Self { lines, selected: 0, offset: 0 } }
    pub fn move_down(&mut self) { if !self.lines.is_empty() { self.selected = (self.selected + 1).min(self.lines.len() - 1); } }
    pub fn move_up(&mut self) { self.selected = self.selected.saturating_sub(1); }
    pub fn visible(&self, height: usize) -> &[DiffLine] { let start = self.offset.min(self.lines.len()); let end = (start + height).min(self.lines.len()); &self.lines[start..end] }
}
#[cfg(test)] mod tests { use super::*; #[test] fn parses_and_navigates_diff() { let mut view = Viewport::from_unified("--- a\n+++ b\n@@\n old\n-new\n+new"); assert_eq!(view.lines.len(), 3); assert_eq!(view.lines[1].kind, DiffKind::Removed); view.move_down(); assert_eq!(view.selected, 1); view.move_up(); assert_eq!(view.selected, 0); assert_eq!(view.visible(2).len(), 2); } }
