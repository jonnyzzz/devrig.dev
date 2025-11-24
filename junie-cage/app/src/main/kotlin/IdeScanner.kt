package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.CIO
import io.ktor.client.request.*
import io.ktor.client.statement.*
import kotlinx.coroutines.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

@Serializable
data class IdeInfo(
    val name: String,
    val version: String,
    val buildNumber: String,
    val configPath: String,
    val pluginsPath: String,
    val systemPath: String,
    val logsPath: String,
    val port: Int
)

class IdeScanner {
    private val client = HttpClient(CIO) {
        expectSuccess = false
        engine {
            requestTimeout = 2000
        }
    }
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")
    private val json = Json { ignoreUnknownKeys = true }

    // IntelliJ-based IDEs port ranges:
    // - Built-in web server: 63342 (default, configurable starting from 1024)
    // - Socket lock ports: 6942-6991 (inter-process communication, not REST API)
    // We scan the built-in web server port which exposes REST API
    //
    // Known REST API endpoints on port 63342:
    // - /api/about - Returns IDE information (name, version, build number)
    // - /api/file - Opens files in the IDE (supports paths, line, column)
    //
    // Note: IntelliJ REST API is implemented via org.jetbrains.ide.RestService
    // and com.intellij.httpRequestHandler extension point. Documentation is limited.
    private val commonPorts = (63342..63362).toList()

    suspend fun scanForIdes(): List<IdeInfo> {
        log("Starting IDE scan on common ports...")
        val foundIdes = mutableListOf<IdeInfo>()

        // Scan ports in parallel for speed
        coroutineScope {
            commonPorts.map { port ->
                async(Dispatchers.IO) {
                    checkPort(port)
                }
            }.awaitAll().filterNotNull().forEach { foundIdes.add(it) }
        }

        log("IDE scan complete. Found ${foundIdes.size} IDE(s)")
        foundIdes.forEach { ide ->
            log("  - ${ide.name} ${ide.version} on port ${ide.port}")
        }

        return foundIdes
    }

    private suspend fun checkPort(port: Int): IdeInfo? {
        try {
            // Try to get IDE information from the REST API
            // Response format: { "name": "PhpStorm 2022.3.1", "productName": "PhpStorm",
            //                    "baselineVersion": 223, "buildNumber": "223.8214.64" }
            val aboutUrl = "http://localhost:$port/api/about"
            val response: HttpResponse = client.get(aboutUrl)

            if (response.status.value in 200..299) {
                val body = response.bodyAsText()
                log("Port $port responded: $body")

                // Try to parse as JSON
                return try {
                    parseIdeInfo(body, port)
                } catch (e: Exception) {
                    log("Failed to parse IDE info from port $port: ${e.message}")
                    // If parsing fails, create basic info with default paths
                    val paths = inferStandardPaths("IntelliJ", "Unknown")
                    IdeInfo(
                        name = "IntelliJ-based IDE",
                        version = "Unknown",
                        buildNumber = "Unknown",
                        configPath = paths.configPath,
                        pluginsPath = paths.pluginsPath,
                        systemPath = paths.systemPath,
                        logsPath = paths.logsPath,
                        port = port
                    )
                }
            }
        } catch (e: Exception) {
            // Port not accessible or no IDE running
        }
        return null
    }

    private fun parseIdeInfo(jsonString: String, port: Int): IdeInfo {
        // Parse IDE information from /api/about JSON response
        // Expected format: { "name": "PhpStorm 2022.3.1", "productName": "PhpStorm",
        //                    "baselineVersion": 223, "buildNumber": "223.8214.64" }
        val jsonElement = json.parseToJsonElement(jsonString)
        val jsonObject = jsonElement.jsonObject

        val name = jsonObject["name"]?.jsonPrimitive?.content
            ?: jsonObject["productName"]?.jsonPrimitive?.content
            ?: "IntelliJ-based IDE"
        val productName = jsonObject["productName"]?.jsonPrimitive?.content ?: name
        val buildNumber = jsonObject["buildNumber"]?.jsonPrimitive?.content ?: "Unknown"

        // Extract version from name (e.g., "PhpStorm 2022.3.1" -> "2022.3.1")
        val version = name.substringAfterLast(" ", "Unknown")

        // Infer standard paths based on OS and IDE name
        // IntelliJ REST API doesn't expose these paths, so we use OS conventions
        val paths = inferStandardPaths(productName, version)

        return IdeInfo(
            name = productName,
            version = version,
            buildNumber = buildNumber,
            configPath = paths.configPath,
            pluginsPath = paths.pluginsPath,
            systemPath = paths.systemPath,
            logsPath = paths.logsPath,
            port = port
        )
    }

    private data class IdePaths(
        val configPath: String,
        val pluginsPath: String,
        val systemPath: String,
        val logsPath: String
    )

    private fun inferStandardPaths(productName: String, version: String): IdePaths {
        // Standard JetBrains IDE directory patterns per OS (version 2020.1+)
        // Reference: https://www.jetbrains.com/help/idea/directories-used-by-the-ide-to-store-settings-caches-plugins-and-logs.html
        val homeDir = System.getProperty("user.home")
        val osName = System.getProperty("os.name").lowercase()

        // Format product name for directory (e.g., "IntelliJIdea2024.2", "PhpStorm2023.3")
        // Pattern: <ProductName><YYYY>.<Minor> where YYYY is year, Minor is release number
        val versionParts = version.split(".")
        val majorMinorVersion = if (versionParts.size >= 2) {
            "${versionParts[0]}.${versionParts[1]}"
        } else {
            version
        }
        val productDir = productName.replace(" ", "") + majorMinorVersion

        return when {
            osName.contains("mac") -> {
                // macOS paths (2020.1+)
                IdePaths(
                    configPath = "$homeDir/Library/Application Support/JetBrains/$productDir",
                    pluginsPath = "$homeDir/Library/Application Support/JetBrains/$productDir",
                    systemPath = "$homeDir/Library/Caches/JetBrains/$productDir",
                    logsPath = "$homeDir/Library/Logs/JetBrains/$productDir"
                )
            }
            osName.contains("linux") -> {
                // Linux paths (2020.1+)
                IdePaths(
                    configPath = "$homeDir/.config/JetBrains/$productDir",
                    pluginsPath = "$homeDir/.local/share/JetBrains/$productDir",
                    systemPath = "$homeDir/.cache/JetBrains/$productDir",
                    logsPath = "$homeDir/.cache/JetBrains/$productDir/log"
                )
            }
            osName.contains("windows") -> {
                // Windows paths (2020.1+)
                val appData = System.getenv("APPDATA") ?: "$homeDir\\AppData\\Roaming"
                val localAppData = System.getenv("LOCALAPPDATA") ?: "$homeDir\\AppData\\Local"
                IdePaths(
                    configPath = "$appData\\JetBrains\\$productDir",
                    pluginsPath = "$appData\\JetBrains\\$productDir",
                    systemPath = "$localAppData\\JetBrains\\$productDir",
                    logsPath = "$localAppData\\JetBrains\\$productDir\\log"
                )
            }
            else -> {
                // Unknown OS, use generic paths
                IdePaths(
                    configPath = "$homeDir/.jetbrains/$productDir/config",
                    pluginsPath = "$homeDir/.jetbrains/$productDir/plugins",
                    systemPath = "$homeDir/.jetbrains/$productDir/system",
                    logsPath = "$homeDir/.jetbrains/$productDir/log"
                )
            }
        }
    }

    private fun log(message: String) {
        val timestamp = LocalDateTime.now().format(timeFormatter)
        println("[$timestamp] [IDE-SCAN] $message")
    }

    fun close() {
        client.close()
    }
}
