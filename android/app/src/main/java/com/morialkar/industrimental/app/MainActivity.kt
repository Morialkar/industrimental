package com.morialkar.industrimental.app

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import com.morialkar.industrimental.mobile.EbitenView

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
    }

    private val ebitenView: EbitenView
        get() = findViewById(R.id.ebitenview)

    override fun onPause() {
        super.onPause()
        ebitenView.suspendGame()
    }

    override fun onResume() {
        super.onResume()
        ebitenView.resumeGame()
    }
}
