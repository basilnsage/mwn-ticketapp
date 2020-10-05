import axios from 'axios';
import { useState } from 'react';

const named = ({url, method, body, onSuccess}) => {
    const [errors, setErrors] = useState(null);

    const doRequest = async () => {
        try {
            setErrors(null);
            const resp = await axios[method](url, body);
            if (onSuccess) {
                onSuccess(resp.data);
            }
            return resp.data;
        } catch(err) {
            setErrors(
                <div className="alert alert-danger">
                <h4>Ooops....</h4>
                <ul className="my-0">
                    <li>{err.response.data.error}</li>
                </ul>
                </div>
            );
        };
    };
    return { doRequest, errors };
};

export default named;