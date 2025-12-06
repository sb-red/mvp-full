package com.softgate;

import com.google.gson.Gson;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.params.SetParams;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Worker {
    private static final String REDIS_HOST = System.getenv().getOrDefault("REDIS_HOST", "localhost");
    private static final int REDIS_PORT = Integer.parseInt(System.getenv().getOrDefault("REDIS_PORT", "6379"));
    private static final String QUEUE_KEY = "execution_queue:java11";
    private static final String RESULT_KEY_PREFIX = "result:";
    private static final int RESULT_TTL = 600; // 10 minutes
    private static final Gson gson = new Gson();

    public static void main(String[] args) {
        System.out.println("Java 11 Worker started. Connecting to Redis at " + REDIS_HOST + ":" + REDIS_PORT);

        try (Jedis jedis = new Jedis(REDIS_HOST, REDIS_PORT)) {
            jedis.ping(); // Test connection
            System.out.println("Connected to Redis successfully");

            while (true) {
                try {
                    // Blocking pop from queue (5 second timeout)
                    List<String> result = jedis.brpop(5, QUEUE_KEY);

                    if (result != null && result.size() == 2) {
                        String rawData = result.get(1);
                        Long invocationId = null;

                        try {
                            Map<String, Object> data = gson.fromJson(rawData, Map.class);
                            invocationId = ((Number) data.get("invocationId")).longValue();
                            String code = (String) data.get("code");
                            @SuppressWarnings("unchecked")
                            Map<String, Object> inputData = (Map<String, Object>) data.getOrDefault("input", Map.of());

                            System.out.println("Processing invocation: " + invocationId);

                            long startTime = System.currentTimeMillis();
                            ExecutionResult execResult = Sandbox.runCode(code, inputData);
                            long duration = System.currentTimeMillis() - startTime;

                            // Parse output as JSON if possible
                            Object outputJson = null;
                            String errorMessage = "";
                            if ("SUCCESS".equals(execResult.status)) {
                                try {
                                    outputJson = gson.fromJson(execResult.output, Object.class);
                                } catch (Exception e) {
                                    outputJson = Map.of("result", execResult.output);
                                }
                            } else {
                                errorMessage = execResult.output;
                            }

                            Map<String, Object> resultMap = new HashMap<>();
                            resultMap.put("invocationId", invocationId);
                            resultMap.put("status", execResult.status);
                            if (outputJson != null) {
                                resultMap.put("output", outputJson);
                            }
                            resultMap.put("outputRaw", execResult.output);
                            if (errorMessage != null && !errorMessage.isEmpty()) {
                                resultMap.put("errorMessage", errorMessage);
                            }
                            resultMap.put("logs", execResult.logs);
                            resultMap.put("durationMs", duration);

                            String resultJson = gson.toJson(resultMap);
                            jedis.set(RESULT_KEY_PREFIX + invocationId, resultJson, SetParams.setParams().ex(RESULT_TTL));

                            System.out.println("Finished invocation: " + invocationId + " - " + execResult.status);

                        } catch (Exception parseError) {
                            System.err.println("Error parsing job: " + parseError.getMessage());
                            parseError.printStackTrace();

                            if (invocationId != null) {
                                Map<String, Object> errorResult = new HashMap<>();
                                errorResult.put("invocationId", invocationId);
                                errorResult.put("status", "ERROR");
                                errorResult.put("output", null);
                                errorResult.put("outputRaw", "");
                                errorResult.put("errorMessage", parseError.getMessage());
                                errorResult.put("logs", getStackTrace(parseError));
                                errorResult.put("durationMs", 0);
                                String errorJson = gson.toJson(errorResult);
                                jedis.set(RESULT_KEY_PREFIX + invocationId, errorJson, SetParams.setParams().ex(RESULT_TTL));
                            }
                        }
                    }
                } catch (Exception err) {
                    System.err.println("Error in worker loop: " + err.getMessage());
                    err.printStackTrace();
                    Thread.sleep(1000); // Small delay before retrying
                }
            }
        } catch (Exception e) {
            System.err.println("Fatal error: " + e.getMessage());
            e.printStackTrace();
        }
    }

    private static String getStackTrace(Exception e) {
        StringBuilder sb = new StringBuilder();
        sb.append(e.toString()).append("\n");
        for (StackTraceElement element : e.getStackTrace()) {
            sb.append("\tat ").append(element.toString()).append("\n");
        }
        return sb.toString();
    }
}
