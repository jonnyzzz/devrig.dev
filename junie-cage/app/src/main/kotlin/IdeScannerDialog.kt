package org.jonnyzzz.ai.app

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import kotlinx.coroutines.launch
import java.awt.Desktop
import java.io.File
import java.net.URI

/**
 * IDE Scanner Dialog - Debug/Test Application
 *
 * This standalone dialog application for testing IDE scanner functionality:
 * 1. Scans for IntelliJ-based IDEs on ports 63342-63362
 * 2. Displays found IDEs with their information
 * 3. Shows system paths (config, plugins, system, logs)
 * 4. Makes all paths and API URLs clickable
 *
 * Usage: Run this main() function to open the IDE scanner dialog
 */
fun main() = application {
    val windowState = rememberWindowState(width = 900.dp, height = 700.dp)

    Window(
        onCloseRequest = ::exitApplication,
        title = "IDE Scanner Dialog",
        state = windowState
    ) {
        MaterialTheme(typography = typographyWithJetBrainsMono()) {
            IdeScannerDialogApp()
        }
    }
}

@Composable
fun IdeScannerDialogApp() {
    var ides by remember { mutableStateOf<List<IdeInfo>>(emptyList()) }
    var isScanning by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val scanner = remember { IdeScanner() }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color(0xFFF5F5F5))
            .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Header
        Text(
            text = "IntelliJ IDE Scanner Test",
            style = MaterialTheme.typography.headlineMedium,
            fontSize = 28.sp,
            color = Color(0xFF3C3F41)
        )

        Text(
            text = "Scans ports 63342-63352 for IntelliJ-based IDEs using REST API",
            style = MaterialTheme.typography.bodyMedium,
            color = Color(0xFF6B6B6B)
        )

        HorizontalDivider(color = Color(0xFFD0D0D0), thickness = 1.dp)

        // Scan button
        Button(
            onClick = {
                scope.launch {
                    isScanning = true
                    error = null
                    try {
                        ides = scanner.scanForIdes()
                    } catch (e: Exception) {
                        error = "Scan failed: ${e.message}"
                    } finally {
                        isScanning = false
                    }
                }
            },
            enabled = !isScanning,
            colors = ButtonDefaults.buttonColors(
                containerColor = Color(0xFF4A9EFF)
            ),
            modifier = Modifier.height(48.dp)
        ) {
            Text(
                text = if (isScanning) "Scanning..." else "Scan for IDEs",
                fontSize = 18.sp
            )
        }

        // Error message
        error?.let { errorMsg ->
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = Color(0xFFFFEBEE)
                ),
                shape = RoundedCornerShape(8.dp)
            ) {
                Text(
                    text = errorMsg,
                    modifier = Modifier.padding(16.dp),
                    color = Color(0xFFC62828)
                )
            }
        }

        // Results
        if (ides.isEmpty() && !isScanning && error == null) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = Color.White
                ),
                shape = RoundedCornerShape(8.dp)
            ) {
                Text(
                    text = "No IDEs scanned yet. Click 'Scan for IDEs' to start.",
                    modifier = Modifier.padding(16.dp),
                    color = Color(0xFF6B6B6B)
                )
            }
        } else if (ides.isEmpty() && !isScanning) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = Color(0xFFFFF9C4)
                ),
                shape = RoundedCornerShape(8.dp)
            ) {
                Text(
                    text = "No running IDEs found. Make sure an IntelliJ-based IDE is running.",
                    modifier = Modifier.padding(16.dp),
                    color = Color(0xFFF57F17)
                )
            }
        } else if (ides.isNotEmpty()) {
            Text(
                text = "Found ${ides.size} IDE(s):",
                style = MaterialTheme.typography.titleMedium,
                fontSize = 20.sp,
                color = Color(0xFF3C3F41)
            )

            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                ides.forEach { ide ->
                    IdeInfoCard(ide)
                }
            }
        }
    }
}

@Composable
fun IdeInfoCard(ide: IdeInfo) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = Color.White
        ),
        shape = RoundedCornerShape(12.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // IDE name header
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = ide.name,
                    style = MaterialTheme.typography.titleLarge,
                    fontSize = 22.sp,
                    color = Color(0xFF3C3F41)
                )

                Surface(
                    color = Color(0xFF4A9EFF),
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Text(
                        text = "Port ${ide.port}",
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        color = Color.White,
                        fontSize = 14.sp
                    )
                }
            }

            HorizontalDivider(color = Color(0xFFE0E0E0))

            // Details
            InfoRow("Version:", ide.version)
            InfoRow("Build Number:", ide.buildNumber)

            HorizontalDivider(color = Color(0xFFE0E0E0), modifier = Modifier.padding(vertical = 8.dp))

            Text(
                text = "File System Paths:",
                fontSize = 14.sp,
                color = Color(0xFF3C3F41),
                modifier = Modifier.padding(bottom = 4.dp)
            )

            ClickablePathRow("Config:", ide.configPath)
            ClickablePathRow("Plugins:", ide.pluginsPath)
            ClickablePathRow("System:", ide.systemPath)
            ClickablePathRow("Logs:", ide.logsPath)

            HorizontalDivider(color = Color(0xFFE0E0E0), modifier = Modifier.padding(vertical = 8.dp))

            // API endpoint - clickable
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "API:",
                    fontSize = 14.sp,
                    color = Color(0xFF6B6B6B),
                    modifier = Modifier.width(120.dp)
                )
                Text(
                    text = "http://localhost:${ide.port}/api/about",
                    fontSize = 14.sp,
                    color = Color(0xFF4A9EFF),
                    textDecoration = TextDecoration.Underline,
                    modifier = Modifier.clickable {
                        openInBrowser("http://localhost:${ide.port}/api/about")
                    }
                )
            }
        }
    }
}

@Composable
fun InfoRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = label,
            fontSize = 14.sp,
            color = Color(0xFF6B6B6B),
            modifier = Modifier.width(120.dp)
        )
        Text(
            text = value,
            fontSize = 14.sp,
            color = Color(0xFF3C3F41)
        )
    }
}

@Composable
fun ClickablePathRow(label: String, path: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = label,
            fontSize = 14.sp,
            color = Color(0xFF6B6B6B),
            modifier = Modifier.width(120.dp)
        )
        Text(
            text = path,
            fontSize = 13.sp,
            color = Color(0xFF4A9EFF),
            textDecoration = TextDecoration.Underline,
            modifier = Modifier
                .weight(1f)
                .clickable {
                    openInFileManager(path)
                }
        )
    }
}

fun openInBrowser(url: String) {
    try {
        if (Desktop.isDesktopSupported()) {
            Desktop.getDesktop().browse(URI(url))
        }
    } catch (e: Exception) {
        println("Failed to open URL: $url - ${e.message}")
    }
}

fun openInFileManager(path: String) {
    try {
        val file = File(path)
        if (Desktop.isDesktopSupported()) {
            if (file.exists()) {
                Desktop.getDesktop().open(file)
            } else {
                println("Path does not exist: $path")
            }
        }
    } catch (e: Exception) {
        println("Failed to open path: $path - ${e.message}")
    }
}
