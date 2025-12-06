package com.softgate;

public class ExecutionResult {
    public final String status;
    public final String output;
    public final String logs;

    public ExecutionResult(String status, String output, String logs) {
        this.status = status;
        this.output = output;
        this.logs = logs;
    }
}
