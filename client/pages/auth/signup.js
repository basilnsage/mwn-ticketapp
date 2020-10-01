import { useState } from 'react';
import axios from 'axios';
import protobuf from 'protobufjs';

const protoDef = '/protoSrc/signin.proto';

function defaultFunc() {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');

    const onSubmit = async (event) => {
        let protoFile = await protobuf.load(protoDef);
        let proto = protoFile.lookupType('signin.SignIn');
        console.log('proto: ', proto);
        event.persist();
        var payload = {
            username: email,
            password: password,
        };
        proto.verify(payload)
        var signupMsg = proto.create(payload);
        var signupBytes = proto.encode(signupMsg).finish();
        console.log(signupBytes);
        const resp = axios({
            method: 'post',
            url: '/api/users/signup',
            headers: {'Content-Type': 'application/octet-stream'},
            data: signupBytes,
            transformRequest: (data, any, headers) => {
                return data;
            },
        });
        return resp
    }

    return (
        <form onSubmit={onSubmit}>
            <script async="async" src="https://cdn.jsdelivr.net/npm/protobufjs@6.10.1/dist/protobuf.js"></script>
            <h1>Sign Up</h1>
            <div className="form-group">
                <label>Email Address</label>
                <input value={email} onChange={e => setEmail(e.target.value)} className="form-control" />
            </div>
            <div className="form-group">
                <label>Password</label>
                <input value={password} onChange={e => setPassword(e.target.value)} type="password" className="form-control" />
            </div>
            <button className="btn btn-primary">Sign Up</button>
        </form>
    );
};

export default defaultFunc;