{
  lib,
  buildGoModule,
  version ? "dev",
  src ? null,
}:

buildGoModule {
  pname = "win-automation";
  inherit version src;

  # Vendor directory committed to repo - no hash needed
  vendorHash = null;

  subPackages = [ "cmd/win-automation" ];

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  # Skip tests that require network/SSH
  doCheck = false;

  meta = with lib; {
    description = "CLI for automating a Windows VM from Linux via SSH and HTTP APIs";
    homepage = "https://github.com/alejg/win-automation";
    license = licenses.mit;
    maintainers = [ ];
    mainProgram = "win-automation";
    platforms = platforms.linux;
  };
}
