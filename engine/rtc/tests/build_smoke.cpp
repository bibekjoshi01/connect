#include <iostream>

// Public engine headers must coexist with the POSIX socket API used by UDP.
#if defined(__unix__) || defined(__APPLE__)
#include <sys/socket.h>
#endif

#include "connect/rtc/build_info.hpp"

int main() {
  const auto version = connect_rtc::build_version();
  if (version.empty()) {
    std::cerr << "Missing native library build metadata\n";
    return 1;
  }
  std::cout << "connect_rtc " << version << '\n';
}
