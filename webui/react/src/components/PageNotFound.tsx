import React from 'react';

import Button from 'components/kit/Button';
import Link from 'components/Link';
import css from 'components/PageNotFound.module.scss';
import { paths } from 'routes/utils';

const PageNotFound: React.FC = () => (
  <div className={css.base}>
    <div className={css.status}>404</div>
    <div>{"Page not found or you don't have access"}</div>
    <div className={css.content}>
      {'Make sure you have the right url or that you have access to view.'}
    </div>
    <Link path={paths.dashboard()}>
      <Button>Back to Home</Button>
    </Link>
  </div>
);

export default PageNotFound;
