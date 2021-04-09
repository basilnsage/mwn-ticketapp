import 'bootstrap/dist/css/bootstrap.css';
import buildClient from '../api/build-client';
import SigninHeader from '../components/header';

const appDefault = ({Component, pageProps, email}) => {
    return (
        <div>
            <SigninHeader email={email}/>
            <Component {...pageProps} />
        </div>
    );
};

appDefault.getInitialProps = async (appContext) => {
    const client = buildClient(appContext.ctx);
    const { data } = await client.get('/api/users/whoami');
    let pageProps = {};
    if (appContext.Component.getInitialProps) {
        pageProps = await appContext.Component.getInitialProps(appContext.ctx);
    };
    return {
        pageProps,
        ...data,
    };
};

export default appDefault;