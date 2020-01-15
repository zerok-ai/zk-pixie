#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <istream>
#include <memory>
#include <sstream>

#include "src/common/fs/fs_wrapper.h"
#include "src/common/system/config_mock.h"
#include "src/common/system/proc_parser.h"
#include "src/common/testing/testing.h"

namespace pl {
namespace system {

using std::string;
using ::testing::Return;

constexpr char kTestDataBasePath[] = "src/common/system";

namespace {
string GetPathToTestDataFile(const string& fname) {
  return TestEnvironment::PathToTestDataFile(std::string(kTestDataBasePath) + "/" + fname);
}
}  // namespace

class ProcParserTest : public ::testing::Test {
 protected:
  void SetUp() override {
    system::MockConfig sysconfig;

    EXPECT_CALL(sysconfig, HasConfig()).WillRepeatedly(Return(true));
    EXPECT_CALL(sysconfig, PageSize()).WillRepeatedly(Return(4096));
    EXPECT_CALL(sysconfig, KernelTicksPerSecond()).WillRepeatedly(Return(10000000));
    EXPECT_CALL(sysconfig, ClockRealTimeOffset()).WillRepeatedly(Return(128));
    EXPECT_CALL(sysconfig, proc_path())
        .WillRepeatedly(Return(GetPathToTestDataFile("testdata/proc")));
    parser_ = std::make_unique<ProcParser>(sysconfig);
    bytes_per_page_ = sysconfig.PageSize();
  }

  std::unique_ptr<ProcParser> parser_;
  int bytes_per_page_ = 0;
};

TEST_F(ProcParserTest, ParseNetworkStat) {
  ProcParser::NetworkStats stats;
  PL_CHECK_OK(parser_->ParseProcPIDNetDev(123, &stats));

  // The expeted values are from the test file above.
  EXPECT_EQ(54504114, stats.rx_bytes);
  EXPECT_EQ(65296, stats.rx_packets);
  EXPECT_EQ(0, stats.rx_drops);
  EXPECT_EQ(0, stats.rx_errs);

  EXPECT_EQ(4258632, stats.tx_bytes);
  EXPECT_EQ(39739, stats.tx_packets);
  EXPECT_EQ(0, stats.tx_drops);
  EXPECT_EQ(0, stats.tx_errs);
}

TEST_F(ProcParserTest, ParseStatIO) {
  ProcParser::ProcessStats stats;
  PL_CHECK_OK(parser_->ParseProcPIDStatIO(123, &stats));

  // The expeted values are from the test file above.
  EXPECT_EQ(5405203, stats.rchar_bytes);
  EXPECT_EQ(1239158, stats.wchar_bytes);
  EXPECT_EQ(17838080, stats.read_bytes);
  EXPECT_EQ(634880, stats.write_bytes);
}

TEST_F(ProcParserTest, ParsePidStat) {
  ProcParser::ProcessStats stats;
  PL_CHECK_OK(parser_->ParseProcPIDStat(123, &stats));

  // The expeted values are from the test file above.
  EXPECT_EQ("ibazel", stats.process_name);

  EXPECT_EQ(800, stats.utime_ns);
  EXPECT_EQ(2300, stats.ktime_ns);
  EXPECT_EQ(13, stats.num_threads);

  EXPECT_EQ(55, stats.major_faults);
  EXPECT_EQ(1799, stats.minor_faults);

  EXPECT_EQ(114384896, stats.vsize_bytes);
  EXPECT_EQ(2577 * bytes_per_page_, stats.rss_bytes);
}

TEST_F(ProcParserTest, ParseStat) {
  ProcParser::SystemStats stats;
  PL_CHECK_OK(parser_->ParseProcStat(&stats));

  // The expected values are from the test file above.
  EXPECT_EQ(248758, stats.cpu_utime_ns);
  EXPECT_EQ(78314, stats.cpu_ktime_ns);
}

TEST_F(ProcParserTest, ParseMemInfo) {
  ProcParser::SystemStats stats;
  auto test_file = GetPathToTestDataFile("testdata/proc/sample_proc_meminfo");
  PL_CHECK_OK(parser_->ParseProcMemInfo(&stats));

  // The expected values are from the test file above.
  EXPECT_EQ(67228110848, stats.mem_total_bytes);
  EXPECT_EQ(17634656256, stats.mem_free_bytes);
  EXPECT_EQ(51960180736, stats.mem_available_bytes);

  EXPECT_EQ(6654636032, stats.mem_buffer_bytes);
  EXPECT_EQ(25549463552, stats.mem_cached_bytes);
  EXPECT_EQ(24576, stats.mem_swap_cached_bytes);

  EXPECT_EQ(28388524032, stats.mem_active_bytes);
  EXPECT_EQ(15734595584, stats.mem_inactive_bytes);
}

TEST_F(ProcParserTest, read_pid_start_time) {
  EXPECT_EQ(14329, parser_->GetPIDStartTimeTicks(123));
}

TEST_F(ProcParserTest, read_pid_cmdline) {
  EXPECT_THAT("/usr/lib/slack/slack --force-device-scale-factor=1.5 --high-dpi-support=1",
              parser_->GetPIDCmdline(123));
}

TEST_F(ProcParserTest, read_pid_metadata_null) {
  EXPECT_THAT("/usr/lib/at-spi2-core/at-spi2-registryd --use-gnome-session",
              parser_->GetPIDCmdline(456));
}

// This test does not work because bazel uses symlinks itself,
// which then causes ReadProcPIDFDLink to resolve the wrong link.
TEST_F(ProcParserTest, read_proc_fd_link) {
  {
    // Bazel doesn't copy symlink testdata as symlinks, so we create the missing symlink testdata
    // here.
    ASSERT_OK(
        fs::CreateSymlinkIfNotExists("/dev/null", GetPathToTestDataFile("testdata/proc/123/fd/0")));
    ASSERT_OK(
        fs::CreateSymlinkIfNotExists("/foobar", GetPathToTestDataFile("testdata/proc/123/fd/1")));
    ASSERT_OK(fs::CreateSymlinkIfNotExists("socket:[12345]",
                                           GetPathToTestDataFile("testdata/proc/123/fd/2")));
  }

  std::string out;
  Status s;

  s = parser_->ReadProcPIDFDLink(123, 0, &out);
  EXPECT_OK(s);
  EXPECT_EQ("/dev/null", out);

  s = parser_->ReadProcPIDFDLink(123, 1, &out);
  EXPECT_OK(s);
  EXPECT_EQ("/foobar", out);

  s = parser_->ReadProcPIDFDLink(123, 2, &out);
  EXPECT_OK(s);
  EXPECT_EQ("socket:[12345]", out);

  s = parser_->ReadProcPIDFDLink(123, 3, &out);
  EXPECT_NOT_OK(s);
}

}  // namespace system
}  // namespace pl
