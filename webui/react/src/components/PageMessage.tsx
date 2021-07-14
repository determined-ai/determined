import React from 'react';

import Logo, { LogoTypes } from 'components/Logo';
import Page from 'components/Page';

import css from './PageMessage.module.scss';

interface Props {
  message?: string;
  title: string;
}

const PageMessage: React.FC<Props> = ({ message, title }: Props) => {
  return(
    <Page docTitle={title}>
      <div className={css.base}>
        <div className={css.content}>
          <Logo type={LogoTypes.OnLightVertical} />
          <p>{message}</p>
        </div>
      </div>
    </Page>
  );
};

export default PageMessage;
