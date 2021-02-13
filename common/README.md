### Common logic used for the mwn-ticketapp application

#### Protos
Define inter-service paylooads
To recompile the Go-specific proto definitions, run
`protoc --proto_path=protos/ --go_opt=module=github.com/basilnsage/mwn-ticketapp/common --go_out=. protos/*.proto`
