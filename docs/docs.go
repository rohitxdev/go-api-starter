// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/": {
            "get": {
                "description": "Home page.",
                "summary": "Home Page",
                "responses": {
                    "200": {
                        "description": "home page",
                        "schema": {
                            "type": "html"
                        }
                    }
                }
            }
        },
        "/_": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Admin route.",
                "summary": "Admin route",
                "responses": {
                    "200": {
                        "description": "Admin page",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "401": {
                        "description": "invalid session",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/config": {
            "get": {
                "description": "Get client config.",
                "summary": "Get config",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/config.Client"
                        }
                    }
                }
            }
        },
        "/ping": {
            "get": {
                "description": "Ping the server.",
                "summary": "Ping",
                "responses": {
                    "200": {
                        "description": "pong",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "config.Client": {
            "type": "object",
            "required": [
                "env"
            ],
            "properties": {
                "env": {
                    "type": "string",
                    "enum": [
                        "development",
                        "production"
                    ]
                },
                "isDev": {
                    "type": "boolean"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
