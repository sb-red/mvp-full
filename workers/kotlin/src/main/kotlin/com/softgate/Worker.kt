package com.softgate

import com.google.gson.Gson
import redis.clients.jedis.Jedis
import redis.clients.jedis.params.SetParams

const val REDIS_HOST = "localhost"
const val REDIS_PORT = 6379
const val QUEUE_KEY = "execution_queue:kotlin"
const val RESULT_KEY_PREFIX = "result:"
const val RESULT_TTL = 600L

fun main() {
    val redisHost = System.getenv("REDIS_HOST") ?: REDIS_HOST
    val redisPort = System.getenv("REDIS_PORT")?.toIntOrNull() ?: REDIS_PORT

    println("Kotlin Worker started. Connecting to Redis at $redisHost:$redisPort")

    val gson = Gson()

    Jedis(redisHost, redisPort).use { jedis ->
        jedis.ping()
        println("Connected to Redis successfully")

        while (true) {
            try {
                // Blocking pop from queue (5 second timeout)
                val result = jedis.brpop(5, QUEUE_KEY)

                if (result != null && result.size == 2) {
                    val rawData = result[1]
                    var invocationId: Long? = null

                    try {
                        val data = gson.fromJson(rawData, Map::class.java) as Map<*, *>
                        invocationId = (data["invocationId"] as? Number)?.toLong()
                        val code = data["code"] as? String ?: throw IllegalArgumentException("Missing code field")
                        @Suppress("UNCHECKED_CAST")
                        val inputData = (data["input"] as? Map<String, Any>) ?: emptyMap()

                        println("Processing invocation: $invocationId")

                        val startTime = System.currentTimeMillis()
                        val execResult = runCode(code, inputData)
                        val duration = System.currentTimeMillis() - startTime

                        // Parse output as JSON if possible
                        val outputJson: Any? = if (execResult.status == "SUCCESS") {
                            try {
                                gson.fromJson(execResult.output, Any::class.java)
                            } catch (e: Exception) {
                                mapOf("result" to execResult.output)
                            }
                        } else {
                            null
                        }

                        val errorMessage = if (execResult.status != "SUCCESS") execResult.output else ""

                        val resultMap = mapOf(
                            "invocationId" to invocationId,
                            "status" to execResult.status,
                            "output" to outputJson,
                            "outputRaw" to execResult.output,
                            "errorMessage" to errorMessage,
                            "logs" to execResult.logs,
                            "durationMs" to duration
                        )

                        val resultJson = gson.toJson(resultMap)
                        jedis.set("$RESULT_KEY_PREFIX$invocationId", resultJson, SetParams.setParams().ex(RESULT_TTL))

                        println("Finished invocation: $invocationId - ${execResult.status}")

                    } catch (e: Exception) {
                        System.err.println("Error parsing job: ${e.message}")
                        e.printStackTrace()

                        invocationId?.let { id ->
                            val errorResult = mapOf(
                                "invocationId" to id,
                                "status" to "ERROR",
                                "output" to null,
                                "outputRaw" to "",
                                "errorMessage" to e.message,
                                "logs" to e.stackTraceToString(),
                                "durationMs" to 0L
                            )
                            val errorJson = gson.toJson(errorResult)
                            jedis.set("$RESULT_KEY_PREFIX$id", errorJson, SetParams.setParams().ex(RESULT_TTL))
                        }
                    }
                }
            } catch (e: Exception) {
                System.err.println("Error in worker loop: ${e.message}")
                e.printStackTrace()
                Thread.sleep(1000) // Small delay before retrying
            }
        }
    }
}
