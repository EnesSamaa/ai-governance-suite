//! Streaming masking for common PII and bearer secrets.

const EMAIL_MASK: &str = "[EMAIL_REDACTED]";
const TCKN_MASK: &str = "[TCKN_REDACTED]";
const CARD_MASK: &str = "[CARD_REDACTED]";
const SECRET_MASK: &str = "[SECRET_REDACTED]";

/// Incremental masker. Input chunks are retained only until their token boundary,
/// allowing values split across network frames to be recognized.
#[derive(Default)]
pub struct StreamMasker {
    pending: String,
}

impl StreamMasker {
    pub fn new() -> Self {
        Self::default()
    }

    /// Masks complete tokens from `chunk`, retaining a trailing partial token.
    pub fn push(&mut self, chunk: &str) -> String {
        self.pending.push_str(chunk);
        let boundary = self
            .pending
            .char_indices()
            .rev()
            .find_map(|(index, character)| (!is_token_char(character)).then_some(index + character.len_utf8()))
            .unwrap_or(0);
        if boundary == 0 {
            return String::new();
        }
        let complete = self.pending[..boundary].to_owned();
        self.pending.drain(..boundary);
        mask_text(&complete)
    }

    /// Flushes the final token and resets the masker for reuse.
    pub fn finish(&mut self) -> String {
        let remaining = std::mem::take(&mut self.pending);
        mask_text(&remaining)
    }
}

/// Masks emails, valid TCKN values, payment card numbers and common API-key forms.
pub fn mask_text(input: &str) -> String {
    input
        .split_inclusive(|character: char| !is_token_char(character))
        .map(|part| {
            let token_end = part.find(|character: char| !is_token_char(character)).unwrap_or(part.len());
            let (token, suffix) = part.split_at(token_end);
            let replacement = classify(token).unwrap_or(token);
            let mut value = String::with_capacity(replacement.len() + suffix.len());
            value.push_str(replacement);
            value.push_str(suffix);
            value
        })
        .collect()
}

fn is_token_char(character: char) -> bool {
    character.is_ascii_alphanumeric() || matches!(character, '@' | '.' | '_' | '-' | '=')
}

fn classify(token: &str) -> Option<&'static str> {
    if is_email(token) { Some(EMAIL_MASK) }
    else if is_valid_tckn(token) { Some(TCKN_MASK) }
    else if is_card_number(token) { Some(CARD_MASK) }
    else if is_secret(token) { Some(SECRET_MASK) }
    else { None }
}

fn is_email(token: &str) -> bool {
    let Some((local, domain)) = token.split_once('@') else { return false };
    !local.is_empty()
        && domain.contains('.')
        && !domain.starts_with('.')
        && !domain.ends_with('.')
        && token.matches('@').count() == 1
}

fn is_valid_tckn(token: &str) -> bool {
    if token.len() != 11 || !token.bytes().all(|value| value.is_ascii_digit()) || token.starts_with('0') {
        return false;
    }
    let digits: Vec<u32> = token.bytes().map(|value| u32::from(value - b'0')).collect();
    let tenth = (((digits[0] + digits[2] + digits[4] + digits[6] + digits[8]) as i32 * 7
        - (digits[1] + digits[3] + digits[5] + digits[7]) as i32).rem_euclid(10)) as u32;
    let eleventh = digits[..10].iter().sum::<u32>() % 10;
    digits[9] == tenth && digits[10] == eleventh
}

fn is_card_number(token: &str) -> bool {
    let digits: String = token.chars().filter(|character| character.is_ascii_digit()).collect();
    if !(13..=19).contains(&digits.len()) || digits.len() != token.len() {
        return false;
    }
    let mut sum = 0;
    for (index, digit) in digits.bytes().rev().enumerate() {
        let mut value = u32::from(digit - b'0');
        if index % 2 == 1 {
            value *= 2;
            if value > 9 { value -= 9; }
        }
        sum += value;
    }
    sum % 10 == 0
}

fn is_secret(token: &str) -> bool {
    (token.starts_with("sk-") && token.len() >= 20)
        || (token.starts_with("ghp_") && token.len() >= 20)
        || (token.starts_with("AKIA") && token.len() == 20)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn masks_sensitive_values() {
        let input = "mail ada@example.com tckn 10000000146 card 4242424242424242 key sk-abcdefghijklmnopqrstu";
        assert_eq!(mask_text(input), "mail [EMAIL_REDACTED] tckn [TCKN_REDACTED] card [CARD_REDACTED] key [SECRET_REDACTED]");
    }

    #[test]
    fn preserves_invalid_lookalikes() {
        assert_eq!(mask_text("not-an-email@ 10000000145 4242424242424241"), "not-an-email@ 10000000145 4242424242424241");
    }

    #[test]
    fn recognizes_values_split_between_chunks() {
        let mut masker = StreamMasker::new();
        assert_eq!(masker.push("contact ada@exam"), "contact ");
        assert_eq!(masker.push("ple.com now "), "[EMAIL_REDACTED] now ");
        assert_eq!(masker.push("sk-abcdefghijklmnopq"), "");
        assert_eq!(masker.finish(), "[SECRET_REDACTED]");
    }
}
