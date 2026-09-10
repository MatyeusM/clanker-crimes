//! Number format service. Formats computed values for display, trimming
//! trailing zeros. Extracted verbatim from main.rs. The Big Function.
pub fn format_number(n: f64) -> String {
  if !n.is_finite() {
    return "Error".to_string();
  }
  if n.abs() < 1e-10 {
    return "0".to_string();
  }
  if n.fract().abs() < 1e-10 && n.abs() < 1e15 {
    format!("{}", n as i64)
  } else {
    let s = format!("{:.10}", n);
    s.trim_end_matches('0').trim_end_matches('.').to_string()
  }
}
