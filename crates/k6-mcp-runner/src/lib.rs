//! Safe k6 command integration and JSON metric aggregation for MCP tools.

use std::collections::BTreeMap;
use std::process::Command;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct RunRequest { pub script: String, pub vus: u32, pub duration: String }
#[derive(Clone, Debug, PartialEq)]
pub struct RunReport { pub success: bool, pub metrics: BTreeMap<String, f64>, pub latency_ms: Option<f64>, pub throughput_rps: Option<f64>, pub stderr: String }
pub trait Executor { fn execute(&self, program: &str, args: &[String]) -> Result<Execution, String>; }
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Execution { pub success: bool, pub stdout: String, pub stderr: String }
pub struct SystemExecutor;
impl Executor for SystemExecutor { fn execute(&self, program: &str, args: &[String]) -> Result<Execution, String> { let output = Command::new(program).args(args).output().map_err(|error| error.to_string())?; Ok(Execution { success: output.status.success(), stdout: String::from_utf8_lossy(&output.stdout).into_owned(), stderr: String::from_utf8_lossy(&output.stderr).into_owned() }) } }
pub struct K6Runner<E> { executor: E }
impl<E: Executor> K6Runner<E> {
    pub fn new(executor: E) -> Self { Self { executor } }
    pub fn run(&self, request: &RunRequest) -> Result<RunReport, String> {
        if request.script.is_empty() || request.vus == 0 || request.duration.is_empty() { return Err("script, vus and duration are required".into()); }
        let args = vec!["run".into(), "--vus".into(), request.vus.to_string(), "--duration".into(), request.duration.clone(), "--out".into(), "json=-".into(), request.script.clone()];
        let execution = self.executor.execute("k6", &args)?;
        let samples = parse_metric_samples(&execution.stdout);
        let metrics = summarize_metrics(&samples);
        let latency_ms = samples.get("http_req_duration").map(|values| values.iter().sum::<f64>() / values.len() as f64);
        let duration_seconds = duration_seconds(&request.duration);
        let throughput_rps = samples.get("http_reqs").and_then(|values| duration_seconds.map(|seconds| values.iter().sum::<f64>() / seconds));
        Ok(RunReport { success: execution.success, metrics, latency_ms, throughput_rps, stderr: execution.stderr })
    }
}
pub fn parse_metrics(output: &str) -> BTreeMap<String, f64> { summarize_metrics(&parse_metric_samples(output)) }
fn parse_metric_samples(output: &str) -> BTreeMap<String, Vec<f64>> { output.lines().filter_map(|line| { let metric = json_field(line, "metric")?; let value = json_field(line, "value")?.parse().ok()?; Some((metric, value)) }).fold(BTreeMap::new(), |mut metrics, (metric, value)| { metrics.entry(metric).or_default().push(value); metrics }) }
fn summarize_metrics(samples: &BTreeMap<String, Vec<f64>>) -> BTreeMap<String, f64> { samples.iter().filter_map(|(metric, values)| (!values.is_empty()).then_some((metric.clone(), values.iter().sum::<f64>() / values.len() as f64))).collect() }
fn duration_seconds(duration: &str) -> Option<f64> { let (number, unit) = duration.split_at(duration.find(|character: char| !character.is_ascii_digit() && character != '.').unwrap_or(duration.len())); let value: f64 = number.parse().ok()?; match unit { "ms" => Some(value / 1000.0), "s" => Some(value), "m" => Some(value * 60.0), _ => None } }
fn json_field(line: &str, field: &str) -> Option<String> { let needle = format!("\"{field}\":"); let after = line.split_once(&needle)?.1.trim_start(); if let Some(value) = after.strip_prefix('"') { Some(value.split('"').next()?.to_owned()) } else { Some(after.split(|character| character == ',' || character == '}').next()?.trim().to_owned()) } }
#[cfg(test)]
mod tests { use super::*; struct Fake; impl Executor for Fake { fn execute(&self, _: &str, args: &[String]) -> Result<Execution, String> { assert!(args.contains(&"--vus".into())); Ok(Execution { success: true, stdout: "{\"metric\":\"http_req_duration\",\"value\":42.5}\n{\"metric\":\"checks\",\"value\":1}".into(), stderr: String::new() }) } }
    #[test] fn runs_k6_and_aggregates_metrics() { let runner = K6Runner::new(Fake); let report = runner.run(&RunRequest { script: "load.js".into(), vus: 5, duration: "30s".into() }).unwrap(); assert!(report.success); assert_eq!(report.metrics["http_req_duration"], 42.5); assert_eq!(report.latency_ms, Some(42.5)); }
    #[test] fn derives_throughput_from_request_samples() { let output = "{\"metric\":\"http_reqs\",\"value\":1}\n{\"metric\":\"http_reqs\",\"value\":1}"; let samples = parse_metric_samples(output); assert_eq!(samples["http_reqs"].iter().sum::<f64>() / duration_seconds("2s").unwrap(), 1.0); }
    #[test] fn rejects_invalid_request() { assert!(K6Runner::new(Fake).run(&RunRequest { script: "".into(), vus: 0, duration: "".into() }).is_err()); }
}
