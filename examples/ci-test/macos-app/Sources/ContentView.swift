import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "checkmark.seal.fill")
                .font(.system(size: 64))
            Text("HexSign CI Test")
                .font(.title).bold()
            Text("Signed and notarized in CI with material fetched by the HexSign CLI.")
                .multilineTextAlignment(.center)
        }
        .padding(40)
        .frame(minWidth: 360, minHeight: 240)
    }
}
