import Button from 'determined-ui/Button';
import Message from 'determined-ui/Message';
import React from 'react';

import { paths } from 'routes/utils';

import Link from './Link';
import css from './PageNotFound.module.scss';

const PageNotFound: React.FC = () => (
  <div className={css.base}>
    <Message
      action={
        <Link path={paths.dashboard()}>
          <Button>Back to Home</Button>
        </Link>
      }
      description="Make sure you have the right url or that you have access to view."
      icon={<div className={css.status}>404</div>}
      title="Page not found or you don't have access"
    />
  </div>
);

export default PageNotFound;
