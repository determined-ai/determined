import React, { PropsWithChildren } from 'react';

import Logo, { Orientation } from 'components/Logo';
import Page from 'components/Page';
import { useStore } from 'contexts/Store';

import css from './PageMessage.module.scss';

interface Props extends PropsWithChildren<unknown> {
  title: string;
}

const PageMessage: React.FC<Props> = ({ title, children }: Props) => {
  const { info } = useStore();
  return(
    <Page docTitle={title}>
      <div className={css.base}>
        <div className={css.content}>
          <Logo branding={info.branding} orientation={Orientation.Vertical} />
          {children}
        </div>
      </div>
    </Page>
  );
};

export default PageMessage;
