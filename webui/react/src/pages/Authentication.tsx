import axios from 'axios';
import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoTypes } from 'components/Logo';
import Spinner from 'components/Spinner';
import Auth, { updateAuth } from 'contexts/Auth';
import { routeAll } from 'routes';
import { defaultAppRoute } from 'routes';
import history from 'routes/history';
import { logout } from 'services/api';

import css from './Authentication.module.scss';

interface Queries {
  redirect?: string;
  cli?: boolean;
}
const Authentication: React.FC = () => {
  const location = useLocation();
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
  const [ isLoading, setIsLoading ] = useState(true);

  const queries: Queries = queryString.parse(location.search);

  const isLogout = location.pathname.endsWith('logout');

  if (isLogout) {
    logout({});
    setAuth({ type: Auth.ActionType.Reset });
    history.push('/det/login' + location.search);
  }

  useEffect(() => {
    const source = axios.CancelToken.source();
    updateAuth(setAuth, source.token).then(() => setIsLoading(false));
    return (): void => {
      source.cancel();
    };
  }, [ setAuth ]);

  if (auth.isAuthenticated) {
    const redirect = queries.redirect || defaultAppRoute.path;
    if (queries.cli) {
      return <AuthToken />;
    }
    routeAll(redirect);
    return <Spinner fullPage />;
  }

  if (isLogout || isLoading) {
    return <Spinner fullPage />;
  }

  return (
    <div className={css.base}>
      <div className={css.content} style={{ display: isLoading ? 'none' : 'inherit' }}>
        <Logo className={css.logo} type={LogoTypes.Dark} />
        <DeterminedAuth setIsLoading={setIsLoading} />
      </div>
    </div>
  );
};

export default Authentication;
