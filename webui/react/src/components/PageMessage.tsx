import React, { PropsWithChildren } from 'react';

import Logo, { LogoTypes } from 'components/Logo';
import Page from 'components/Page';

import css from './PageMessage.module.scss';

interface Props extends PropsWithChildren<unknown> {
  title: string;
}

const PageMessage: React.FC<Props> = ({ title, children }: Props) => {
  return(
    <Page docTitle={title}>
      <div className={css.base}>
        <div className={css.content}>
          <Logo type={LogoTypes.OnLightVertical} />
          {children}
        </div>
      </div>
    </Page>
  );
};

export default PageMessage;
