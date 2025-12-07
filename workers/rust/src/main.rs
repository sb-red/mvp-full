mod executor;

use redis::Commands;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::env;
use std::time::Instant;

const QUEUE_KEY: &str = "execution_queue:rust";
const RESULT_KEY_PREFIX: &str = "result:";
const RESULT_TTL_SECONDS: u64 = 600;

#[derive(Debug, Deserialize)]
struct ExecutionRequest {
    #[serde(rename = "invocationId")]
    invocation_id: i64,
    code: String,
    #[serde(default)]
    input: Value,
}

#[derive(Debug, Serialize)]
struct ExecutionResult {
    #[serde(rename = "invocationId")]
    invocation_id: i64,
    status: String,
    output: Option<Value>,
    #[serde(rename = "outputRaw")]
    output_raw: String,
    #[serde(rename = "errorMessage")]
    error_message: String,
    logs: String,
    #[serde(rename = "durationMs")]
    duration_ms: u64,
}

fn main() {
    let redis_host = env::var("REDIS_HOST").unwrap_or_else(|_| "localhost".to_string());
    let redis_port = env::var("REDIS_PORT").unwrap_or_else(|_| "6379".to_string());

    let redis_url = format!("redis://{}:{}", redis_host, redis_port);
    eprintln!("Rust Worker started. Connecting to Redis at {}:{}", redis_host, redis_port);

    let client = match redis::Client::open(redis_url.as_str()) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("Failed to create Redis client: {}", e);
            return;
        }
    };

    let mut con = match client.get_connection() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("Failed to connect to Redis: {}", e);
            return;
        }
    };

    eprintln!("Connected to Redis successfully");

    loop {
        // Block and wait for job from queue (timeout 5 seconds)
        let result: Option<(String, String)> = con
            .brpop(QUEUE_KEY, 5.0)
            .unwrap_or(None);

        if let Some((_, raw_data)) = result {
            let req: ExecutionRequest = match serde_json::from_str(&raw_data) {
                Ok(r) => r,
                Err(e) => {
                    eprintln!("Error parsing request JSON: {}", e);
                    continue;
                }
            };

            eprintln!("Processing invocation: {}", req.invocation_id);

            let start_time = Instant::now();

            // Convert input Value to a map for executor
            let input_map = match req.input {
                Value::Object(m) => m.into_iter().collect(),
                _ => std::collections::HashMap::new(),
            };

            let (status, output, logs) = executor::run_code(&req.code, input_map);
            let duration_ms = start_time.elapsed().as_millis() as u64;

            let (output_parsed, error_message) = if status == "SUCCESS" {
                let parsed = if !output.is_empty() {
                    serde_json::from_str(&output).ok()
                } else {
                    None
                };
                (parsed, String::new())
            } else {
                (None, output.clone())
            };

            let exec_result = ExecutionResult {
                invocation_id: req.invocation_id,
                status: status.clone(),
                output: output_parsed,
                output_raw: output,
                error_message,
                logs,
                duration_ms,
            };

            let result_json = serde_json::to_string(&exec_result)
                .expect("Failed to serialize result");

            let result_key = format!("{}{}", RESULT_KEY_PREFIX, req.invocation_id);
            let _: () = con
                .set_ex(&result_key, &result_json, RESULT_TTL_SECONDS)
                .expect("Failed to store result");

            eprintln!("Finished invocation: {} - {}", req.invocation_id, status);
        }
    }
}
