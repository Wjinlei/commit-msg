package validator

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/JayceChant/commit-msg/state"
)

const (
	mergePrefix   = "Merge "
	revertPattern = `^(Revert|revert)(:| ).+`
	headerPattern = `^((fixup! |squash! )?(\w+)(?:\(([^\)\s]+)\))?: (.+))(?:\n|$)`
)

// Validate ...
func Validate(file string) {
	defer func() {
		err := recover()
		state, ok := err.(state.State)
		if !ok {
			panic(err)
		}

		if state.IsNormal() {
			os.Exit(0)
		} else {
			os.Exit(int(state))
		}
	}()

	validateMsg(getMsg(file), Config)
}

func getMsg(path string) string {
	if path == "" {
		state.ArgumentMissing.LogAndExit()
	}

	f, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
		state.FileMissing.LogAndExit(path)
	}

	if f.IsDir() {
		log.Println(path, "is not a file.")
		state.FileMissing.LogAndExit(path)
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		state.ReadError.LogAndExit(path)
	}

	return string(buf)
}

func validateMsg(msg string, config *globalConfig) {
	if isEmpty(msg) {
		state.EmptyMessage.LogAndExit()
	}

	isMergeCommit(msg)

	sections := strings.SplitN(msg, "\n", 2)

	if config.LineLimit <= 0 {
		config.LineLimit = 80
	}

	checkHeader(sections[0], config)

	if len(sections) == 2 {
		checkBody(sections[1], config)
	} else if config.BodyRequired {
		state.BodyMissing.LogAndExit()
	}

	state.Validated.LogAndExit()
}

func isEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

func isMergeCommit(msg string) {
	if strings.HasPrefix(msg, mergePrefix) {
		state.Merge.LogAndExit()
	}
}

func checkHeader(header string, config *globalConfig) {
	if isEmpty(header) {
		state.EmptyHeader.LogAndExit()
	}

	if isRevertHeader(header) {
		// skip revert header checking
		return
		// but later body check is still required
	}

	re := regexp.MustCompile(headerPattern)
	groups := re.FindStringSubmatch(header)

	if groups == nil || isEmpty(groups[5]) {
		state.BadHeaderFormat.LogAndExit(header)
	}

	typ := groups[3]
	checkType(typ)

	isFixupOrSquash := (groups[2] != "")

	checkScope(groups[4], config)

	// TODO: 根据规则对subject检查
	// subject := groups[5]

	length := len(header)
	if length > config.LineLimit &&
		!(isFixupOrSquash || typ == "revert" || typ == "Revert") {
		state.LineOverLong.LogAndExit(length, config.LineLimit, header)
	}
}

func isRevertHeader(header string) bool {
	m, _ := regexp.MatchString(revertPattern, header)
	return m
}

func checkType(typ string) {
	for t := range TypeSet {
		if typ == t {
			return
		}
	}
	state.WrongType.LogAndExit(typ, TypesStr)
}

func checkScope(scope string, config *globalConfig) {
	if isEmpty(scope) {
		if config.ScopeRequired {
			state.ScopeMissing.LogAndExit()
		}
		return
	}

	if len(config.Scopes) == 0 {
		return
	}

	for _, s := range config.Scopes {
		if scope == s {
			return
		}
	}
	state.WrongScope.LogAndExit(scope, strings.Join(config.Scopes, ", "))
}

func checkBody(body string, config *globalConfig) {
	if isEmpty(body) {
		if config.BodyRequired {
			state.BodyMissing.LogAndExit()
		} else {
			state.Validated.LogAndExit()
		}
	}

	if !isEmpty(strings.SplitN(body, "\n", 2)[0]) {
		state.NoBlankLineBeforeBody.LogAndExit()
	}

	for _, line := range strings.Split(body, "\n") {
		length := len(line)
		if length > config.LineLimit {
			state.LineOverLong.LogAndExit(length, config.LineLimit, line)
		}
	}
}
