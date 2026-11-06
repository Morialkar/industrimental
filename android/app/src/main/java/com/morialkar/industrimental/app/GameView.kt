package com.morialkar.industrimental.app

import android.content.Context
import android.util.AttributeSet
import com.morialkar.industrimental.mobile.EbitenView

/** Hosts the Go game; the place to hook crash reporting later. */
class GameView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
) : EbitenView(context, attrs) {
    override fun onErrorOnGameUpdate(e: Exception) {
        super.onErrorOnGameUpdate(e)
    }
}
