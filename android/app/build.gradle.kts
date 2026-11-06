plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.morialkar.industrimental.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.morialkar.industrimental"
        minSdk = 24
        targetSdk = 35
        versionCode = 1
        versionName = "0.0.1"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    // Built by: ebitenmobile bind -target android -javapkg com.morialkar.industrimental -o android/app/libs/industrimental.aar ./mobile
    implementation(files("libs/industrimental.aar"))
    implementation("androidx.appcompat:appcompat:1.7.0")
}
