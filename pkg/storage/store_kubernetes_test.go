package storage

import (
    "crypto/md5"
    "encoding/hex"
    "fmt"
    "math/rand"
    "strconv"
    "strings"
    "testing"
)

func Test_getFromKey(t *testing.T) {
    tests := []struct {
        key string
        ok bool
    }{
        {"", false},
        {"a", false},
        {"a/a", true},
        {"/", false},
        {"/a", false},
        {"a/", false},
        {"a////a", false},
        {"a/a/a", false},
        {"/a/a", false},
        {"啊啊啊/啊啊啊", true},
        {"u🎈、/u🎆", true},
        {"コンバース/コンバース", true},
    }

    for _, tt := range tests {
        testName := fmt.Sprintf("Parse \"%s\"", tt.key)
        t.Run(testName, func(t *testing.T) {
            name, resource, err := getFromKey(tt.key)
            if err == nil && tt.ok {
                t.Logf("name: %s, resource: %s", name, resource)
            }
            if (err == nil && tt.ok == false) || (err != nil && tt.ok == true) {
                t.Errorf("Test failed, err: %v, tt.ok: %v", err, tt.ok)
            }
        })
    }
}

func BenchmarkConvert(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := "啊/啊 啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊"
        name := strings.Replace(s, "/", ".", -1)
        name = strconv.QuoteToASCII(name)
        name = name[1: len(name)-1]
        name = strings.Replace(name, "\\", "", -1)
        name = strings.Replace(name, " ", "", -1)
    }
}


func BenchmarkMD5(b *testing.B) {
    for i := 0; i < b.N; i++ {
        name := "啊/啊 啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊"
        m := md5.Sum([]byte(name))
        name = hex.EncodeToString(m[:])
    }
}

func BenchmarkReplace(b *testing.B) {
    for i := 0; i < b.N; i++ {
        name := "啊/啊 啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊啊"
        name = strings.Replace(name, "/", ".", -1)
    }
}

func BenchmarkIf(b *testing.B) {
    m := [...]string{PropertySetTypes, ThingTypes, Things, AgentTypes, Agents, Mappings, EventTypes, Rules, Installations, Recipients, MessageTmpls, Templates, Messages, Servers}
    for i := 0; i < b.N; i++ {
        r := rand.Int() % 14
        n := m[r]

        if n == PropertySetTypes || n == AgentTypes || n == ThingTypes {
            r = 0
        } else {
            r = 1
        }
    }
}

func BenchmarkSwitch(b *testing.B) {
    m := [...]string{PropertySetTypes, ThingTypes, Things, AgentTypes, Agents, Mappings, EventTypes, Rules, Installations, Recipients, MessageTmpls, Templates, Messages, Servers}
    for i := 0; i < b.N; i++ {
        r := rand.Int() % 14
        n := m[r]

        switch n {
        case PropertySetTypes, AgentTypes, ThingTypes:
            r = 0
        default:
            r = 1
        }
    }
}