package org.jonnyzzz.ai.app

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.toComposeImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.platform.Font
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.*
import org.jetbrains.skia.Image as SkiaImage
import java.awt.Toolkit

// Shared dialog components and utilities

@Composable
fun BoxScope.ConnectionNotification(onAccept: () -> Unit) {
    Card(
        modifier = Modifier
            .wrapContentHeight()
            .padding(16.dp)
            .shadow(
                elevation = 16.dp,
                shape = RoundedCornerShape(16.dp),
                ambientColor = Color.Black.copy(alpha = 0.2f),
                spotColor = Color.Black.copy(alpha = 0.2f)
            )
            .clickable { onAccept() },
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFFF5F5F5)
        ),
        border = androidx.compose.foundation.BorderStroke(2.dp, Color(0xFFD0D0D0))
    ) {
        Column(
            modifier = Modifier.fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Header with darker background
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color(0xFF3C3F41))
                    .padding(horizontal = 40.dp, vertical = 24.dp),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "Nvidia GDX Spark Connected",
                    style = MaterialTheme.typography.titleMedium,
                    fontSize = 30.sp,
                    color = Color(0xFFDFE1E5)
                )
            }

            // Content section
            Column(
                modifier = Modifier
                    .padding(40.dp)
                    .fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(32.dp)
            ) {
                LogosRow()

                Button(
                    onClick = { onAccept() },
                    modifier = Modifier.align(Alignment.End),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFF4A9EFF)
                    ),
                    contentPadding = PaddingValues(horizontal = 32.dp, vertical = 12.dp)
                ) {
                    Text("Start Local AI", fontSize = 22.sp)
                }
            }
        }
    }
}

@Composable
private fun LogosRow() {
    val jetbrains = remember { imageFromResourceOrNull("logos/jetbrains.png") }
    val nvidia = remember { imageFromResourceOrNull("logos/nvidia.png") }

    Row(horizontalArrangement = Arrangement.spacedBy(24.dp), verticalAlignment = Alignment.CenterVertically) {
        LogoBox(jetbrains, fallbackText = "JetBrains")
        Text("×", fontSize = 28.sp, color = Color(0xFF6B6B6B))
        LogoBox(nvidia, fallbackText = "NVIDIA")
    }
}

@Composable
private fun LogoBox(img: ImageBitmap?, fallbackText: String) {
    Box(
        modifier = Modifier
            .size(180.dp)
            .clip(RoundedCornerShape(16.dp))
            .background(Color.White)
            .then(
                Modifier.padding(2.dp)
            ),
        contentAlignment = Alignment.Center
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .clip(RoundedCornerShape(14.dp))
                .background(Color.White)
                .border(
                    width = 2.dp,
                    color = Color(0xFFD0D0D0),
                    shape = RoundedCornerShape(14.dp)
                ),
            contentAlignment = Alignment.Center
        ) {
            if (img != null) {
                Image(
                    bitmap = img,
                    contentDescription = fallbackText,
                    modifier = Modifier.padding(20.dp),
                    contentScale = ContentScale.Fit
                )
            } else {
                Text(fallbackText, fontSize = 20.sp)
            }
        }
    }
}

fun imageFromResourceOrNull(resourcePath: String): ImageBitmap? = try {
    val bytes = object {}.javaClass.classLoader?.getResourceAsStream(resourcePath)?.readBytes()
    bytes?.let { SkiaImage.makeFromEncoded(it).toComposeImageBitmap() }
} catch (t: Throwable) {
    null
}

// Shared font family
@Composable
fun jetbrainsMonoFamily(): FontFamily {
    return remember {
        FontFamily(
            Font(resource = "fonts/JetBrainsMono-Regular.ttf")
        )
    }
}

// Shared typography with JetBrains Mono
@Composable
fun typographyWithJetBrainsMono(): Typography {
    val font = jetbrainsMonoFamily()
    return Typography(
        displayLarge = MaterialTheme.typography.displayLarge.copy(fontFamily = font),
        displayMedium = MaterialTheme.typography.displayMedium.copy(fontFamily = font),
        displaySmall = MaterialTheme.typography.displaySmall.copy(fontFamily = font),
        headlineLarge = MaterialTheme.typography.headlineLarge.copy(fontFamily = font),
        headlineMedium = MaterialTheme.typography.headlineMedium.copy(fontFamily = font),
        headlineSmall = MaterialTheme.typography.headlineSmall.copy(fontFamily = font),
        titleLarge = MaterialTheme.typography.titleLarge.copy(fontFamily = font),
        titleMedium = MaterialTheme.typography.titleMedium.copy(fontFamily = font),
        titleSmall = MaterialTheme.typography.titleSmall.copy(fontFamily = font),
        bodyLarge = MaterialTheme.typography.bodyLarge.copy(fontFamily = font),
        bodyMedium = MaterialTheme.typography.bodyMedium.copy(fontFamily = font),
        bodySmall = MaterialTheme.typography.bodySmall.copy(fontFamily = font),
        labelLarge = MaterialTheme.typography.labelLarge.copy(fontFamily = font),
        labelMedium = MaterialTheme.typography.labelMedium.copy(fontFamily = font),
        labelSmall = MaterialTheme.typography.labelSmall.copy(fontFamily = font)
    )
}

// Shared notification window setup
@Composable
fun ApplicationScope.NotificationWindow(
    title: String = "Spark",
    visible: Boolean = true,
    alwaysOnTop: Boolean = false,
    onCloseRequest: () -> Unit,
    content: @Composable BoxScope.() -> Unit
) {
    val screenSize = Toolkit.getDefaultToolkit().screenSize
    val windowWidth = 800.dp
    val xPos = (screenSize.width * 0.75 - windowWidth.value / 2).toInt()
    val yPos = (screenSize.height * 0.75 - 300).toInt()

    val windowState = rememberWindowState(
        position = WindowPosition(xPos.dp, yPos.dp),
        size = DpSize(windowWidth, 600.dp)
    )

    Window(
        onCloseRequest = onCloseRequest,
        title = title,
        state = windowState,
        undecorated = true,
        transparent = true,
        resizable = false,
        alwaysOnTop = alwaysOnTop,
        visible = visible
    ) {
        MaterialTheme(typography = typographyWithJetBrainsMono()) {
            Box(Modifier.wrapContentSize(), content = content)
        }
    }
}

// Debug main function
fun main() = application {
    NotificationWindow(
        title = "Debug Dialog",
        alwaysOnTop = true,
        onCloseRequest = ::exitApplication
    ) {
        ConnectionNotification(
            onAccept = {
                println("Dialog dismissed!")
                exitApplication()
            }
        )
    }
}
