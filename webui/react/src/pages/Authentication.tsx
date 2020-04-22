import { Button, Form, Input } from 'antd';
import { message } from 'antd';
import React from 'react';
import { useHistory, useLocation } from 'react-router';
import { Redirect } from 'react-router-dom';

import Icon from 'components/Icon';
import Link from 'components/Link';
import Logo, { LogoTypes } from 'components/Logo';
import Spinner from 'components/Spinner';
import Auth from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { login, logout } from 'services/api';
import { Credentials } from 'types';

import css from './Authentication.module.scss';

const DEFAULT_REDIRECT = '/det/dashboard';

const Authentication: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();

  const isLogout = location.pathname.endsWith('logout');
  if (isLogout) {
    logout({}).then(() => {
      setAuth({ type: Auth.ActionType.Reset });
      history.push('/det/login');
    });
    return <Spinner fullPage />;
  }

  const onFinish = (creds: unknown): void => {
    // TODO validate the creds type?
    const hideLoader = message.loading('logging in..');
    login(creds as Credentials)
      .then(() => {
        setAuth({ type: Auth.ActionType.SetIsAuthenticated, value: true });
      })
      .catch((e: Error) => {
        // TODO check for the code or error type?
        handleError({
          error: e,
          isUserTriggered: true,
          message: e.message,
          publicMessage: 'Failed to login. Please check your username and password.',
          publicSubject: 'Login failed',
          type: ErrorType.Input,
        });
      })
      .finally(hideLoader);
  };

  const loginForm = (
    <Form
      initialValues={{
        remember: true,
      }}
      name="basic"
      onFinish={onFinish}
    >
      <Form.Item
        name="username"
        rules={[
          {
            message: 'Please input your username!',
            required: true,
          },
        ]}
      >
        <Input placeholder="Username" prefix={<Icon name="user" />} />
      </Form.Item>

      <Form.Item name="password">
        <Input.Password placeholder="Password" prefix={<Icon name="lock" />} />
      </Form.Item>

      <Form.Item>
        <Button htmlType="submit" type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>

  );

  return (
    <div className={css.base}>
      {auth.isAuthenticated && <Redirect to={DEFAULT_REDIRECT} />}
      <div className={css.content}>
        {/* DISCUSSION what if we didn't need to add the logo classname and was able to
        target logo on its own using component name easily in module.scss */}
        <Logo className={css.logo} type={LogoTypes.Dark} />
        {loginForm}
        <Link path="/docs/system-administration/users.html?highlight=user">
        Forgot your password or need to create a user? Checkout our docs
        </Link>
      </div>
    </div>
  );
};

export default Authentication;
