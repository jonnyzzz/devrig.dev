package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.okhttp.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import kotlinx.coroutines.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import kotlin.time.Duration.Companion.milliseconds

class PingServer(
    private val onServerDetected: () -> Unit
) {
    private val client = HttpClient(OkHttp) {
        engine {
            config {
                followRedirects(true)
                readTimeout(200.milliseconds)
                connectTimeout(100.milliseconds)
                writeTimeout(100.milliseconds)
                callTimeout(150.milliseconds)
            }
        }
        expectSuccess = false
    }

    private var job: Job? = null
    private var wasOnline = true
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        job = CoroutineScope(Dispatchers.IO + CoroutineName("Ping")).launch {
            log("Health check started for ${Config.HEALTH_CHECK_URL}")
            while (isActive) {
                checkHealth()
                delay(Config.HEALTH_CHECK_INTERVAL_MS)
            }
        }
    }

    private suspend fun checkHealth() {
        try {
            val response: HttpResponse = client.get(Config.HEALTH_CHECK_URL)

            if (response.status.value in 200..299) {
                if (!wasOnline) {
                    log("Target server DETECTED at ${Config.HEALTH_CHECK_URL}")
                    wasOnline = true
                    // Notify that server is back online
                    withContext(Dispatchers.Main) {
                        onServerDetected()
                    }
                }
            } else {
                handleOffline("returned status ${response.status.value}")
            }
        } catch (e: Exception) {
            handleOffline(e.message ?: "unknown error")
        }
    }

    private fun handleOffline(reason: String) {
        if (wasOnline) {
            log("Target server LOST at ${Config.HEALTH_CHECK_URL}: $reason")
            wasOnline = false
        }
    }

    private fun log(message: String) {
        val timestamp = LocalDateTime.now().format(timeFormatter)
        println("[$timestamp] [PING] $message")
    }

    fun stop() {
        job?.cancel()
        client.close()
    }
}
