import UIKit

final class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    var window: UIWindow?
    private let gameController = MobileEbitenViewController()

    func scene(
        _ scene: UIScene,
        willConnectTo session: UISceneSession,
        options connectionOptions: UIScene.ConnectionOptions
    ) {
        guard let windowScene = scene as? UIWindowScene else { return }
        if let dir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first {
            MobileSetDataDir(dir.path)
        }
        let window = UIWindow(windowScene: windowScene)
        window.rootViewController = gameController
        window.makeKeyAndVisible()
        self.window = window
    }

    func sceneWillResignActive(_ scene: UIScene) {
        gameController.suspendGame()
        MobileSave()
    }

    func sceneDidBecomeActive(_ scene: UIScene) {
        gameController.resumeGame()
    }
}
