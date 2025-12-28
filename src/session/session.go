package session

import (
	"errors"
	"io"
	"log"
	netsmtp "net/smtp"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/goyal-aman/mailmux/src/types"
	"go.starlark.net/starlark"
)

type Session struct {
	CurrentUser types.UserConfig
	From        string
	To          []string

	cfg types.Config
}

func NewSession(cfg types.Config) *Session {
	return &Session{cfg: cfg}
}

// Data handles the mail transmission
func (s *Session) Data(r io.Reader) error {
	mailData, _ := io.ReadAll(r)

	// 1. Convert Downstreams to Starlark List of Structs
	var starlarkDownstreams []starlark.Value
	for _, ds := range s.CurrentUser.Downstreams {
		dsDict := starlark.NewDict(3)
		dsDict.SetKey(starlark.String("addr"), starlark.String(ds.Addr))
		dsDict.SetKey(starlark.String("user"), starlark.String(ds.User))
		dsDict.SetKey(starlark.String("pass"), starlark.String(ds.Pass))
		starlarkDownstreams = append(starlarkDownstreams, dsDict)
	}
	slDownstreams := starlark.NewList(starlarkDownstreams)

	// 2. Define the send function to be called from Starlark
	// It expects a single argument: the downstream dict
	sendFunc := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var dsDict *starlark.Dict
		if err := starlark.UnpackArgs("send", args, kwargs, "ds", &dsDict); err != nil {
			return nil, err
		}

		// Extract fields from dict
		addrV, _, _ := dsDict.Get(starlark.String("addr"))
		userV, _, _ := dsDict.Get(starlark.String("user"))
		passV, _, _ := dsDict.Get(starlark.String("pass"))

		addr := addrV.(starlark.String).GoString()
		user := userV.(starlark.String).GoString()
		pass := passV.(starlark.String).GoString()

		// Perform the actual SMTP send
		auth := netsmtp.PlainAuth("", user, pass, strings.Split(addr, ":")[0])
		err := netsmtp.SendMail(addr, auth, s.From, s.To, mailData)
		if err != nil {
			return starlark.String(err.Error()), nil // Return error as string
		}
		return starlark.None, nil // Success
	}

	// 3. Setup Starlark Thread & Predeclared environment
	thread := &starlark.Thread{Name: "selector"}
	predeclared := starlark.StringDict{
		"send": starlark.NewBuiltin("send", sendFunc),
	}

	// 4. Execute the user's script
	log.Println("Loading selector algo:", s.CurrentUser.SelectorAlgoPath)
	globals, err := starlark.ExecFile(thread, s.CurrentUser.SelectorAlgoPath, nil, predeclared)
	if err != nil {
		log.Printf("Failed to load selector algo: %v", err)
		// Fallback: try first downstream
		if len(s.CurrentUser.Downstreams) > 0 {
			ds := s.CurrentUser.Downstreams[0]
			auth := netsmtp.PlainAuth("", ds.User, ds.Pass, strings.Split(ds.Addr, ":")[0])
			return netsmtp.SendMail(ds.Addr, auth, s.From, s.To, mailData)
		}
		return err
	}

	// 5. Call the 'selector' function from the script
	selectorFunc, ok := globals["selector"]
	if !ok {
		return errors.New("script must define a 'selector(downstreams)' function")
	}

	// selector(downstreams)
	ret, err := starlark.Call(thread, selectorFunc, starlark.Tuple{slDownstreams}, nil)
	if err != nil {
		return err
	}

	// Check return value
	if ret == starlark.None {
		return nil // Success
	}
	return errors.New(ret.String())
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.From = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.To = append(s.To, to)
	return nil
}

func (s *Session) Reset() {
	s.From = ""
	s.To = nil
}

func (s *Session) Logout() error {
	return nil
}

// 1. The Session must tell the server what it can do
func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

// 2. The Session handles the actual verification
func (s *Session) Auth(mech string) (sasl.Server, error) {
	if mech == sasl.Plain {
		return sasl.NewPlainServer(func(identity, username, password string) error {
			if identity != "" && identity != username {
				return errors.New("identities not supported")
			}
			return s.AuthPlain(username, password)
		}), nil
	}
	return nil, errors.New("unsupported mechanism")
}

// 2. The Session handles the actual verification
// AuthPlain is the specific helper method for PLAIN auth
func (s *Session) AuthPlain(username, password string) error {
	for _, u := range s.cfg.Users {
		if u.Email == username && u.Password == password {
			s.CurrentUser = u // Ensure this is stored to prevent the 'index out of range' panic
			return nil
		}
	}
	return errors.New("Invalid credentials")
}

func (s *Session) AuthLogin(username, password string) error {
	return s.AuthPlain(username, password)
}
