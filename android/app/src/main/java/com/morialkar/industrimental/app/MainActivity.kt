package com.morialkar.industrimental.app

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import com.morialkar.industrimental.mobile.EbitenView
import com.morialkar.industrimental.mobile.Mobile

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Mobile.setDataDir(filesDir.absolutePath)
        setContentView(R.layout.activity_main)
    }

    private val ebitenView: EbitenView
        get() = findViewById(R.id.ebitenview)

    override fun onPause() {
        super.onPause()
        ebitenView.suspendGame()
        Mobile.save()
    }

    override fun onResume() {
        super.onResume()
        ebitenView.resumeGame()
    }
}
