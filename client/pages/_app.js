import 'bootstrap/dist/css/bootstrap.css';
import buildClient from '../api/build-client';

const appDefault = ({Component, pageProps}) => {
    return (
        <Component {...pageProps} />
    );
};

// appDefault.getInitialProps = async (context) => {
    // const client = buildClient(context);
    // const { data } = await client.get('/api/users/whoami');
    // return data;
// }

export default appDefault;