package org.jonnyzzz.ai.app

import androidx.compose.runtime.*
import androidx.compose.ui.window.application

fun main() = application {
    // State to track whether to show dialog
    var showDialog by remember { mutableStateOf(false) }

    // Start proxy server
    val proxyServer = remember {
        ProxyServer().apply { start() }
    }

    // Start ping server with callback to show dialog
    val pingServer = remember {
        PingServer(
            onServerDetected = {
                showDialog = true
            }
        ).apply { start() }
    }

    NotificationWindow(
        visible = showDialog,
        alwaysOnTop = showDialog,
        onCloseRequest = {
            proxyServer.stop()
            pingServer.stop()
            exitApplication()
        }
    ) {
        ConnectionNotification(
            onAccept = {
                showDialog = false
            }
        )
    }
}
