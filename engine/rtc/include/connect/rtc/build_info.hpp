#pragma once

#include <string_view>

namespace connect_rtc {
// Library build metadata. This is not a media protocol or an engine implementation.
[[nodiscard]] std::string_view build_version() noexcept;
}  // namespace connect_rtc
