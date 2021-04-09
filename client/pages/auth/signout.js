import Router from 'next/router';
import { useEffect } from 'react';
import useRequest from '../../hooks/use-request';

const signoutDefault = () => {
    const { doRequest } = useRequest({
        url: '/api/users/signout',
        method: 'get',
        onSuccess: () => {
            Router.push('/')
        },
    });

    useEffect(() => {
        doRequest();
    }, []);
    return <div>Signing out</div>
};

export default signoutDefault;