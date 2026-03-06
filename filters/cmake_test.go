package filters

import (
	"strings"
	"testing"
)

func TestFilterCmake_Configure(t *testing.T) {
	raw := "-- The C compiler identification is GNU 12.3.0\n" +
		"-- Detecting C compiler ABI info\n" +
		"-- Detecting C compiler ABI info - done\n" +
		"-- Configuring done (1.2s)\n" +
		"-- Generating done (0.1s)\n" +
		"-- Build files have been written to: /home/user/project/build\n"

	got, err := filterCmake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "configured in 1.2s") {
		t.Errorf("expected config time, got: %s", got)
	}
}

func TestFilterCmake_Build(t *testing.T) {
	raw := "[ 25%] Building CXX object CMakeFiles/myapp.dir/src/main.cpp.o\n" +
		"[ 50%] Building CXX object CMakeFiles/myapp.dir/src/utils.cpp.o\n" +
		"[ 75%] Building CXX object CMakeFiles/myapp.dir/src/config.cpp.o\n" +
		"[100%] Linking CXX executable myapp\n" +
		"[100%] Built target myapp\n"

	got, err := filterCmake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "built target myapp") {
		t.Errorf("expected target, got: %s", got)
	}
}

func TestFilterCmake_ConfigureRealistic(t *testing.T) {
	raw := "-- The C compiler identification is GNU 13.2.0\n" +
		"-- The CXX compiler identification is GNU 13.2.0\n" +
		"-- Detecting C compiler ABI info\n" +
		"-- Detecting C compiler ABI info - done\n" +
		"-- Check for working C compiler: /usr/bin/cc - skipped\n" +
		"-- Detecting C compile features\n" +
		"-- Detecting C compile features - done\n" +
		"-- Detecting CXX compiler ABI info\n" +
		"-- Detecting CXX compiler ABI info - done\n" +
		"-- Check for working CXX compiler: /usr/bin/c++ - skipped\n" +
		"-- Detecting CXX compile features\n" +
		"-- Detecting CXX compile features - done\n" +
		"-- Found OpenSSL: /usr/lib/x86_64-linux-gnu/libcrypto.so (found version \"3.0.13\")\n" +
		"-- Found ZLIB: /usr/lib/x86_64-linux-gnu/libz.so (found version \"1.2.13\")\n" +
		"-- Found Threads: TRUE\n" +
		"-- Looking for pthread_create in pthreads\n" +
		"-- Looking for pthread_create in pthreads - not found\n" +
		"-- Looking for pthread_create in pthread\n" +
		"-- Looking for pthread_create in pthread - found\n" +
		"-- Found Boost: /usr/lib/x86_64-linux-gnu/cmake/Boost-1.83.0/BoostConfig.cmake (found version \"1.83.0\") found components: filesystem system\n" +
		"-- Configuring done (2.3s)\n" +
		"-- Generating done (0.1s)\n" +
		"-- Build files have been written to: /home/user/project/build\n"

	got, err := filterCmake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "configured in 2.3s") {
		t.Errorf("expected config time 2.3s, got: %s", got)
	}
	if !strings.Contains(got, "output: /home/user/project/build") {
		t.Errorf("expected output dir, got: %s", got)
	}
	if len(got) >= len(strings.TrimSpace(raw)) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
}

func TestFilterCmake_BuildRealistic(t *testing.T) {
	raw := "[  5%] Building CXX object src/CMakeFiles/mylib.dir/core.cpp.o\n" +
		"[ 11%] Building CXX object src/CMakeFiles/mylib.dir/utils.cpp.o\n" +
		"/home/user/project/src/utils.cpp:45:12: warning: unused variable 'temp' [-Wunused-variable]\n" +
		"   45 |     auto temp = std::string{};\n" +
		"      |          ^~~~\n" +
		"[ 16%] Building CXX object src/CMakeFiles/mylib.dir/parser.cpp.o\n" +
		"[ 22%] Building CXX object src/CMakeFiles/mylib.dir/network.cpp.o\n" +
		"[ 27%] Linking CXX shared library libmylib.so\n" +
		"[ 27%] Built target mylib\n" +
		"[ 33%] Building CXX object app/CMakeFiles/myapp.dir/main.cpp.o\n" +
		"[ 38%] Building CXX object app/CMakeFiles/myapp.dir/app.cpp.o\n" +
		"[ 44%] Building CXX object app/CMakeFiles/myapp.dir/config.cpp.o\n" +
		"[ 50%] Linking CXX executable myapp\n" +
		"[ 50%] Built target myapp\n" +
		"[ 55%] Building CXX object tests/CMakeFiles/tests.dir/test_core.cpp.o\n" +
		"[ 61%] Building CXX object tests/CMakeFiles/tests.dir/test_utils.cpp.o\n" +
		"[ 66%] Building CXX object tests/CMakeFiles/tests.dir/test_parser.cpp.o\n" +
		"[ 72%] Building CXX object tests/CMakeFiles/tests.dir/test_network.cpp.o\n" +
		"[ 77%] Linking CXX executable tests\n" +
		"[ 77%] Built target tests\n" +
		"[ 83%] Building CXX object benchmarks/CMakeFiles/bench.dir/bench_core.cpp.o\n" +
		"[ 88%] Building CXX object benchmarks/CMakeFiles/bench.dir/bench_parser.cpp.o\n" +
		"[ 94%] Linking CXX executable bench\n" +
		"[100%] Built target bench\n"

	got, err := filterCmake(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tgt := range []string{"mylib", "myapp", "tests", "bench"} {
		if !strings.Contains(got, "built target "+tgt) {
			t.Errorf("expected target %s, got: %s", tgt, got)
		}
	}
	if !strings.Contains(got, "files compiled)") {
		t.Errorf("expected compiled file count, got: %s", got)
	}
	if len(got) >= len(strings.TrimSpace(raw)) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
}

func TestFilterCmake_Empty(t *testing.T) {
	got, err := filterCmake("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
