import buildClient from '../api/build-client';

const hello = (currentUser) => {
    return currentUser.email ? (
        <h1>Welcome! You are signed in!</h1>
    ) : (
        <h1>Welcome! Please sign in to continue.</h1>
    );
};

hello.getInitialProps = async (context) => {
    const client = buildClient(context);
    const { data } = await client.get('/api/users/whoami');
    return data;
}

export default hello;