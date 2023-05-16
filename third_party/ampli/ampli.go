// ampli.go
//
// Ampli - A strong typed wrapper for your Analytics
//
// This file is generated by Amplitude.
// To update run 'ampli pull ddev'
//
// Required dependencies: github.com/amplitude/analytics-go@latest
// Tracking Plan Version: 4
// Build: 1.0.0
// Runtime: go-ampli
//
// View Tracking Plan: https://data.amplitude.com/ddev/DDEV/events/ampli/latest
//
// Full Setup Instructions: https://data.amplitude.com/ddev/DDEV/implementation/ampli/latest/getting-started/ddev
//

package ampli

import (
	"log"
	"sync"

	"github.com/amplitude/analytics-go/amplitude"
)

type (
	EventOptions  = amplitude.EventOptions
	ExecuteResult = amplitude.ExecuteResult
)

const (
	IdentifyEventType      = amplitude.IdentifyEventType
	GroupIdentifyEventType = amplitude.GroupIdentifyEventType

	ServerZoneUS = amplitude.ServerZoneUS
	ServerZoneEU = amplitude.ServerZoneEU
)

var (
	NewClientConfig = amplitude.NewConfig
	NewClient       = amplitude.NewClient
)

var Instance = Ampli{}

type Environment string

const (
	EnvironmentDevelopment Environment = `development`
)

var APIKey = map[Environment]string{
	EnvironmentDevelopment: ``,
}

// LoadClientOptions is Client options setting to initialize Ampli client.
//
// Params:
//   - APIKey: the API key of Amplitude project
//   - Instance: the core SDK instance used by Ampli client
//   - Configuration: the core SDK client configuration instance
type LoadClientOptions struct {
	APIKey        string
	Instance      amplitude.Client
	Configuration amplitude.Config
}

// LoadOptions is options setting to initialize Ampli client.
//
// Params:
//   - Environment: the environment of Amplitude Data project
//   - Disabled: the flag of disabled Ampli client
//   - Client: the LoadClientOptions struct
type LoadOptions struct {
	Environment Environment
	Disabled    bool
	Client      LoadClientOptions
}

type baseEvent struct {
	eventType  string
	properties map[string]interface{}
}

type Event interface {
	ToAmplitudeEvent() amplitude.Event
}

func newBaseEvent(eventType string, properties map[string]interface{}) baseEvent {
	return baseEvent{
		eventType:  eventType,
		properties: properties,
	}
}

func (event baseEvent) ToAmplitudeEvent() amplitude.Event {
	return amplitude.Event{
		EventType:       event.eventType,
		EventProperties: event.properties,
	}
}

var Identify = struct {
	Builder func() interface {
		DockerPlatform(dockerPlatform string) interface {
			DockerVersion(dockerVersion string) interface {
				Language(language string) interface {
					Os(os string) interface {
						Platform(platform string) interface {
							Timezone(timezone string) interface {
								Version(version string) IdentifyBuilder
							}
						}
					}
				}
			}
		}
	}
}{
	Builder: func() interface {
		DockerPlatform(dockerPlatform string) interface {
			DockerVersion(dockerVersion string) interface {
				Language(language string) interface {
					Os(os string) interface {
						Platform(platform string) interface {
							Timezone(timezone string) interface {
								Version(version string) IdentifyBuilder
							}
						}
					}
				}
			}
		}
	} {
		return &identifyBuilder{
			properties: map[string]interface{}{},
		}
	},
}

type IdentifyEvent interface {
	Event
	identify()
}

type identifyEvent struct {
	baseEvent
}

func (e identifyEvent) identify() {
}

type IdentifyBuilder interface {
	Build() IdentifyEvent
	WslDistro(wslDistro string) IdentifyBuilder
}

type identifyBuilder struct {
	properties map[string]interface{}
}

func (b *identifyBuilder) DockerPlatform(dockerPlatform string) interface {
	DockerVersion(dockerVersion string) interface {
		Language(language string) interface {
			Os(os string) interface {
				Platform(platform string) interface {
					Timezone(timezone string) interface {
						Version(version string) IdentifyBuilder
					}
				}
			}
		}
	}
} {
	b.properties[`Docker Platform`] = dockerPlatform

	return b
}

func (b *identifyBuilder) DockerVersion(dockerVersion string) interface {
	Language(language string) interface {
		Os(os string) interface {
			Platform(platform string) interface {
				Timezone(timezone string) interface {
					Version(version string) IdentifyBuilder
				}
			}
		}
	}
} {
	b.properties[`Docker Version`] = dockerVersion

	return b
}

func (b *identifyBuilder) Language(language string) interface {
	Os(os string) interface {
		Platform(platform string) interface {
			Timezone(timezone string) interface {
				Version(version string) IdentifyBuilder
			}
		}
	}
} {
	b.properties[`Language`] = language

	return b
}

func (b *identifyBuilder) Os(os string) interface {
	Platform(platform string) interface {
		Timezone(timezone string) interface {
			Version(version string) IdentifyBuilder
		}
	}
} {
	b.properties[`OS`] = os

	return b
}

func (b *identifyBuilder) Platform(platform string) interface {
	Timezone(timezone string) interface {
		Version(version string) IdentifyBuilder
	}
} {
	b.properties[`Platform`] = platform

	return b
}

func (b *identifyBuilder) Timezone(timezone string) interface {
	Version(version string) IdentifyBuilder
} {
	b.properties[`Timezone`] = timezone

	return b
}

func (b *identifyBuilder) Version(version string) IdentifyBuilder {
	b.properties[`Version`] = version

	return b
}

func (b *identifyBuilder) WslDistro(wslDistro string) IdentifyBuilder {
	b.properties[`WSL Distro`] = wslDistro

	return b
}

func (b *identifyBuilder) Build() IdentifyEvent {
	return &identifyEvent{
		newBaseEvent(`Identify`, b.properties),
	}
}

func (e identifyEvent) ToAmplitudeEvent() amplitude.Event {
	identify := amplitude.Identify{}
	for name, value := range e.properties {
		identify.Set(name, value)
	}

	return amplitude.Event{
		EventType:      IdentifyEventType,
		UserProperties: identify.Properties,
	}
}

var Command = struct {
	Builder func() interface {
		Arguments(arguments []string) interface {
			CalledAs(calledAs string) interface {
				CommandName(commandName string) interface {
					CommandPath(commandPath string) CommandBuilder
				}
			}
		}
	}
}{
	Builder: func() interface {
		Arguments(arguments []string) interface {
			CalledAs(calledAs string) interface {
				CommandName(commandName string) interface {
					CommandPath(commandPath string) CommandBuilder
				}
			}
		}
	} {
		return &commandBuilder{
			properties: map[string]interface{}{},
		}
	},
}

type CommandEvent interface {
	Event
	command()
}

type commandEvent struct {
	baseEvent
}

func (e commandEvent) command() {
}

type CommandBuilder interface {
	Build() CommandEvent
}

type commandBuilder struct {
	properties map[string]interface{}
}

func (b *commandBuilder) Arguments(arguments []string) interface {
	CalledAs(calledAs string) interface {
		CommandName(commandName string) interface {
			CommandPath(commandPath string) CommandBuilder
		}
	}
} {
	b.properties[`Arguments`] = arguments

	return b
}

func (b *commandBuilder) CalledAs(calledAs string) interface {
	CommandName(commandName string) interface {
		CommandPath(commandPath string) CommandBuilder
	}
} {
	b.properties[`Called As`] = calledAs

	return b
}

func (b *commandBuilder) CommandName(commandName string) interface {
	CommandPath(commandPath string) CommandBuilder
} {
	b.properties[`Command Name`] = commandName

	return b
}

func (b *commandBuilder) CommandPath(commandPath string) CommandBuilder {
	b.properties[`Command Path`] = commandPath

	return b
}

func (b *commandBuilder) Build() CommandEvent {
	return &commandEvent{
		newBaseEvent(`Command`, b.properties),
	}
}

var Project = struct {
	Builder func() interface {
		Containers(containers []string) interface {
			ContainersOmitted(containersOmitted []string) interface {
				FailOnHookFail(failOnHookFail bool) interface {
					Id(id string) interface {
						MutagenEnabled(mutagenEnabled bool) interface {
							NfsMountEnabled(nfsMountEnabled bool) interface {
								NodejsVersion(nodejsVersion string) interface {
									PhpVersion(phpVersion string) interface {
										ProjectType(projectType string) interface {
											RouterDisabled(routerDisabled bool) interface {
												TraefikEnabled(traefikEnabled bool) interface {
													WebserverType(webserverType string) ProjectBuilder
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}{
	Builder: func() interface {
		Containers(containers []string) interface {
			ContainersOmitted(containersOmitted []string) interface {
				FailOnHookFail(failOnHookFail bool) interface {
					Id(id string) interface {
						MutagenEnabled(mutagenEnabled bool) interface {
							NfsMountEnabled(nfsMountEnabled bool) interface {
								NodejsVersion(nodejsVersion string) interface {
									PhpVersion(phpVersion string) interface {
										ProjectType(projectType string) interface {
											RouterDisabled(routerDisabled bool) interface {
												TraefikEnabled(traefikEnabled bool) interface {
													WebserverType(webserverType string) ProjectBuilder
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	} {
		return &projectBuilder{
			properties: map[string]interface{}{},
		}
	},
}

type ProjectEvent interface {
	Event
	project()
}

type projectEvent struct {
	baseEvent
}

func (e projectEvent) project() {
}

type ProjectBuilder interface {
	Build() ProjectEvent
	DatabaseType(databaseType string) ProjectBuilder
	DatabaseVersion(databaseVersion string) ProjectBuilder
}

type projectBuilder struct {
	properties map[string]interface{}
}

func (b *projectBuilder) Containers(containers []string) interface {
	ContainersOmitted(containersOmitted []string) interface {
		FailOnHookFail(failOnHookFail bool) interface {
			Id(id string) interface {
				MutagenEnabled(mutagenEnabled bool) interface {
					NfsMountEnabled(nfsMountEnabled bool) interface {
						NodejsVersion(nodejsVersion string) interface {
							PhpVersion(phpVersion string) interface {
								ProjectType(projectType string) interface {
									RouterDisabled(routerDisabled bool) interface {
										TraefikEnabled(traefikEnabled bool) interface {
											WebserverType(webserverType string) ProjectBuilder
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
} {
	b.properties[`Containers`] = containers

	return b
}

func (b *projectBuilder) ContainersOmitted(containersOmitted []string) interface {
	FailOnHookFail(failOnHookFail bool) interface {
		Id(id string) interface {
			MutagenEnabled(mutagenEnabled bool) interface {
				NfsMountEnabled(nfsMountEnabled bool) interface {
					NodejsVersion(nodejsVersion string) interface {
						PhpVersion(phpVersion string) interface {
							ProjectType(projectType string) interface {
								RouterDisabled(routerDisabled bool) interface {
									TraefikEnabled(traefikEnabled bool) interface {
										WebserverType(webserverType string) ProjectBuilder
									}
								}
							}
						}
					}
				}
			}
		}
	}
} {
	b.properties[`Containers Omitted`] = containersOmitted

	return b
}

func (b *projectBuilder) FailOnHookFail(failOnHookFail bool) interface {
	Id(id string) interface {
		MutagenEnabled(mutagenEnabled bool) interface {
			NfsMountEnabled(nfsMountEnabled bool) interface {
				NodejsVersion(nodejsVersion string) interface {
					PhpVersion(phpVersion string) interface {
						ProjectType(projectType string) interface {
							RouterDisabled(routerDisabled bool) interface {
								TraefikEnabled(traefikEnabled bool) interface {
									WebserverType(webserverType string) ProjectBuilder
								}
							}
						}
					}
				}
			}
		}
	}
} {
	b.properties[`Fail On Hook Fail`] = failOnHookFail

	return b
}

func (b *projectBuilder) Id(id string) interface {
	MutagenEnabled(mutagenEnabled bool) interface {
		NfsMountEnabled(nfsMountEnabled bool) interface {
			NodejsVersion(nodejsVersion string) interface {
				PhpVersion(phpVersion string) interface {
					ProjectType(projectType string) interface {
						RouterDisabled(routerDisabled bool) interface {
							TraefikEnabled(traefikEnabled bool) interface {
								WebserverType(webserverType string) ProjectBuilder
							}
						}
					}
				}
			}
		}
	}
} {
	b.properties[`ID`] = id

	return b
}

func (b *projectBuilder) MutagenEnabled(mutagenEnabled bool) interface {
	NfsMountEnabled(nfsMountEnabled bool) interface {
		NodejsVersion(nodejsVersion string) interface {
			PhpVersion(phpVersion string) interface {
				ProjectType(projectType string) interface {
					RouterDisabled(routerDisabled bool) interface {
						TraefikEnabled(traefikEnabled bool) interface {
							WebserverType(webserverType string) ProjectBuilder
						}
					}
				}
			}
		}
	}
} {
	b.properties[`Mutagen Enabled`] = mutagenEnabled

	return b
}

func (b *projectBuilder) NfsMountEnabled(nfsMountEnabled bool) interface {
	NodejsVersion(nodejsVersion string) interface {
		PhpVersion(phpVersion string) interface {
			ProjectType(projectType string) interface {
				RouterDisabled(routerDisabled bool) interface {
					TraefikEnabled(traefikEnabled bool) interface {
						WebserverType(webserverType string) ProjectBuilder
					}
				}
			}
		}
	}
} {
	b.properties[`NFS Mount Enabled`] = nfsMountEnabled

	return b
}

func (b *projectBuilder) NodejsVersion(nodejsVersion string) interface {
	PhpVersion(phpVersion string) interface {
		ProjectType(projectType string) interface {
			RouterDisabled(routerDisabled bool) interface {
				TraefikEnabled(traefikEnabled bool) interface {
					WebserverType(webserverType string) ProjectBuilder
				}
			}
		}
	}
} {
	b.properties[`Nodejs Version`] = nodejsVersion

	return b
}

func (b *projectBuilder) PhpVersion(phpVersion string) interface {
	ProjectType(projectType string) interface {
		RouterDisabled(routerDisabled bool) interface {
			TraefikEnabled(traefikEnabled bool) interface {
				WebserverType(webserverType string) ProjectBuilder
			}
		}
	}
} {
	b.properties[`PHP Version`] = phpVersion

	return b
}

func (b *projectBuilder) ProjectType(projectType string) interface {
	RouterDisabled(routerDisabled bool) interface {
		TraefikEnabled(traefikEnabled bool) interface {
			WebserverType(webserverType string) ProjectBuilder
		}
	}
} {
	b.properties[`Project Type`] = projectType

	return b
}

func (b *projectBuilder) RouterDisabled(routerDisabled bool) interface {
	TraefikEnabled(traefikEnabled bool) interface {
		WebserverType(webserverType string) ProjectBuilder
	}
} {
	b.properties[`Router Disabled`] = routerDisabled

	return b
}

func (b *projectBuilder) TraefikEnabled(traefikEnabled bool) interface {
	WebserverType(webserverType string) ProjectBuilder
} {
	b.properties[`Traefik Enabled`] = traefikEnabled

	return b
}

func (b *projectBuilder) WebserverType(webserverType string) ProjectBuilder {
	b.properties[`Webserver Type`] = webserverType

	return b
}

func (b *projectBuilder) DatabaseType(databaseType string) ProjectBuilder {
	b.properties[`Database Type`] = databaseType

	return b
}

func (b *projectBuilder) DatabaseVersion(databaseVersion string) ProjectBuilder {
	b.properties[`Database Version`] = databaseVersion

	return b
}

func (b *projectBuilder) Build() ProjectEvent {
	return &projectEvent{
		newBaseEvent(`Project`, b.properties),
	}
}

type Ampli struct {
	Disabled bool
	Client   amplitude.Client
	mutex    sync.RWMutex
}

// Load initializes the Ampli wrapper.
// Call once when your application starts.
func (a *Ampli) Load(options LoadOptions) {
	if a.Client != nil {
		log.Print("Warn: Ampli is already initialized. Ampli.Load() should be called once at application start up.")

		return
	}

	var apiKey string
	switch {
	case options.Client.APIKey != "":
		apiKey = options.Client.APIKey
	case options.Environment != "":
		apiKey = APIKey[options.Environment]
	default:
		apiKey = options.Client.Configuration.APIKey
	}

	if apiKey == "" && options.Client.Instance == nil {
		log.Print("Error: Ampli.Load() requires option.Environment, " +
			"and apiKey from either options.Instance.APIKey or APIKey[options.Environment], " +
			"or options.Instance.Instance")
	}

	clientConfig := options.Client.Configuration

	if clientConfig.Plan == nil {
		clientConfig.Plan = &amplitude.Plan{
			Branch:    `ampli`,
			Source:    `ddev`,
			Version:   `4`,
			VersionID: `da01c7d9-18a0-4c51-bac5-e231d9aac3c6`,
		}
	}

	if clientConfig.IngestionMetadata == nil {
		clientConfig.IngestionMetadata = &amplitude.IngestionMetadata{
			SourceName:    `go-go-ampli`,
			SourceVersion: `2.0.0`,
		}
	}

	if options.Client.Instance != nil {
		a.Client = options.Client.Instance
	} else {
		clientConfig.APIKey = apiKey
		a.Client = amplitude.NewClient(clientConfig)
	}

	a.mutex.Lock()
	a.Disabled = options.Disabled
	a.mutex.Unlock()
}

// InitializedAndEnabled checks if Ampli is initialized and enabled.
func (a *Ampli) InitializedAndEnabled() bool {
	if a.Client == nil {
		log.Print("Error: Ampli is not yet initialized. Have you called Ampli.Load() on app start?")

		return false
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return !a.Disabled
}

func (a *Ampli) setUserID(userID string, eventOptions *EventOptions) {
	if userID != "" {
		eventOptions.UserID = userID
	}
}

// Track tracks an event.
func (a *Ampli) Track(userID string, event Event, eventOptions ...EventOptions) {
	if !a.InitializedAndEnabled() {
		return
	}

	var options EventOptions
	if len(eventOptions) > 0 {
		options = eventOptions[0]
	}

	a.setUserID(userID, &options)

	baseEvent := event.ToAmplitudeEvent()
	baseEvent.EventOptions = options

	a.Client.Track(baseEvent)
}

// Identify identifies a user and set user properties.
func (a *Ampli) Identify(userID string, identify IdentifyEvent, eventOptions ...EventOptions) {
	a.Track(userID, identify, eventOptions...)
}

// Flush flushes events waiting in buffer.
func (a *Ampli) Flush() {
	if !a.InitializedAndEnabled() {
		return
	}

	a.Client.Flush()
}

// Shutdown disables and shutdowns Ampli Instance.
func (a *Ampli) Shutdown() {
	if !a.InitializedAndEnabled() {
		return
	}

	a.mutex.Lock()
	a.Disabled = true
	a.mutex.Unlock()

	a.Client.Shutdown()
}

func (a *Ampli) Command(userID string, event CommandEvent, eventOptions ...EventOptions) {
	a.Track(userID, event, eventOptions...)
}

func (a *Ampli) Project(userID string, event ProjectEvent, eventOptions ...EventOptions) {
	a.Track(userID, event, eventOptions...)
}
