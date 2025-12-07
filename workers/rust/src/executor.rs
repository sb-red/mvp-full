use serde_json::Value;
use std::collections::HashMap;
use std::fs;
use std::io::Write;
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::time::Duration;
use uuid::Uuid;

const COMPILE_TIMEOUT_SECS: u64 = 60;
const EXECUTION_TIMEOUT_SECS: u64 = 30;

fn get_wrapper_code(user_code: &str) -> String {
    format!(
        r#"use serde_json::{{json, Value}};
use std::io::{{self, Read}};

{}

fn main() {{
    let mut input = String::new();
    io::stdin().read_to_string(&mut input).expect("Failed to read stdin");

    let event: Value = serde_json::from_str(&input).expect("Failed to parse input JSON");
    let result = handler(event);

    println!("{{}}", result);
}}
"#,
        user_code
    )
}

fn get_cargo_toml() -> &'static str {
    r#"[package]
name = "handler"
version = "0.1.0"
edition = "2018"

[dependencies]
serde_json = "1.0"
serde = { version = "1.0", features = ["derive"] }
"#
}

pub fn run_code(code: &str, input_data: HashMap<String, Value>) -> (String, String, String) {
    let work_dir = PathBuf::from("/tmp/sandbox").join(Uuid::new_v4().to_string());

    if let Err(e) = fs::create_dir_all(&work_dir) {
        return ("ERROR".to_string(), format!("Failed to create work directory: {}", e), String::new());
    }

    let result = run_code_internal(code, input_data, &work_dir);

    // Cleanup
    let _ = fs::remove_dir_all(&work_dir);

    result
}

fn run_code_internal(code: &str, input_data: HashMap<String, Value>, work_dir: &PathBuf) -> (String, String, String) {
    let mut logs = String::new();

    // Create Cargo project structure
    let src_dir = work_dir.join("src");
    if let Err(e) = fs::create_dir_all(&src_dir) {
        return ("ERROR".to_string(), format!("Failed to create src directory: {}", e), logs);
    }

    // Write Cargo.toml
    let cargo_path = work_dir.join("Cargo.toml");
    if let Err(e) = fs::write(&cargo_path, get_cargo_toml()) {
        return ("ERROR".to_string(), format!("Failed to write Cargo.toml: {}", e), logs);
    }

    // Write main.rs with wrapper
    let main_rs_path = src_dir.join("main.rs");
    let full_code = get_wrapper_code(code);
    if let Err(e) = fs::write(&main_rs_path, &full_code) {
        return ("ERROR".to_string(), format!("Failed to write main.rs: {}", e), logs);
    }

    // Compile with cargo build --release
    let compile_result = Command::new("cargo")
        .args(["build", "--release"])
        .current_dir(work_dir)
        .env("CARGO_HOME", "/usr/local/cargo")
        .env("RUSTUP_HOME", "/usr/local/rustup")
        .output();

    let compile_output = match compile_result {
        Ok(output) => output,
        Err(e) => {
            return ("ERROR".to_string(), format!("Failed to run cargo: {}", e), logs);
        }
    };

    let compile_stderr = String::from_utf8_lossy(&compile_output.stderr).to_string();
    logs.push_str(&compile_stderr);

    if !compile_output.status.success() {
        return ("ERROR".to_string(), format!("Compilation error:\n{}", compile_stderr), logs);
    }

    logs.push_str("Compilation successful\n");

    // Run the compiled binary
    let binary_path = work_dir.join("target/release/handler");

    let input_json = match serde_json::to_string(&Value::Object(input_data.into_iter().collect())) {
        Ok(j) => j,
        Err(e) => {
            return ("ERROR".to_string(), format!("Failed to serialize input: {}", e), logs);
        }
    };

    let mut child = match Command::new(&binary_path)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .current_dir(work_dir)
        .spawn()
    {
        Ok(c) => c,
        Err(e) => {
            return ("ERROR".to_string(), format!("Failed to execute binary: {}", e), logs);
        }
    };

    // Write input to stdin
    if let Some(mut stdin) = child.stdin.take() {
        let _ = stdin.write_all(input_json.as_bytes());
    }

    // Wait with timeout
    let output = match child.wait_with_output() {
        Ok(o) => o,
        Err(e) => {
            return ("TIMEOUT".to_string(), format!("Execution timed out: {}", e), logs);
        }
    };

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    if !stderr.is_empty() {
        logs.push_str(&stderr);
    }

    if !output.status.success() {
        let error_msg = if stderr.is_empty() {
            "Execution failed with non-zero exit code".to_string()
        } else {
            stderr
        };
        return ("ERROR".to_string(), error_msg, logs);
    }

    let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
    ("SUCCESS".to_string(), stdout, logs)
}
