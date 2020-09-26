package main

import (
	"github.com/basilnsage/mwn-ticketapp/common/protos"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
)

func main() {
	user := protos.SignIn{
		Username: "foo@example.com",
		Password: "876543210",
	}
	data, _ := proto.Marshal(&user)
	ioutil.WriteFile("test.proto", data, 0640)
	user = protos.SignIn{
		Username: "foo@@example.com",
		Password: "876543210",
	}
	data, _ = proto.Marshal(&user)
	ioutil.WriteFile("test2.proto", data, 0640)
	user = protos.SignIn{
		Username: "",
		Password: "876543210",
	}
	data, _ = proto.Marshal(&user)
	ioutil.WriteFile("test3.proto", data, 0640)
	user = protos.SignIn{
		Username: "foa@example.com",
		Password: "876543210",
	}
	data, _ = proto.Marshal(&user)
	ioutil.WriteFile("test4.proto", data, 0640)
}
