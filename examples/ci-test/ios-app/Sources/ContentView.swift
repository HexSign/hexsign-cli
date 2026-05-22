import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "checkmark.seal.fill")
                .font(.system(size: 64))
            Text("HexSign CI Test")
                .font(.title2).bold()
            Text("Signed in CI with material fetched by the HexSign CLI.")
                .multilineTextAlignment(.center)
                .padding(.horizontal)
        }
        .padding()
    }
}
