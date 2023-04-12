import React, { useRef } from 'react';

import Icon from 'components/kit/Icon';
import Page from 'components/Page';

import css from './NoPermissions.module.scss';

const NoPermissions: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page bodyNoPadding containerRef={pageRef}>
      <div className={css.base}>
        <div className={css.icon}>
          <Icon name="warning-large" size="mega" />
        </div>
        <h6>You don&lsquo;t have access to a workspace</h6>
        <p className={css.message}>
          In order to access and use Determined you need to be added to a workspace. Contact your
          admin to request to be added to a workspace.
        </p>
      </div>
    </Page>
  );
};

export default NoPermissions;
