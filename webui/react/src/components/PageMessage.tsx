import { useObservable } from 'micro-observables';
import React from 'react';

import Logo, { Orientation } from 'components/Logo';
import Page from 'components/Page';
import determinedStore, { BrandingType } from 'stores/determinedInfo';

import css from './PageMessage.module.scss';

interface Props {
  children: React.ReactNode;
  title: string;
}

const PageMessage: React.FC<Props> = ({ title, children }: Props) => {
  const info = useObservable(determinedStore.info);
  return (
    <Page breadcrumb={[]} docTitle={title} noScroll>
      <div className={css.base}>
        <div className={css.content}>
          <Logo
            branding={info.branding || BrandingType.Determined}
            orientation={Orientation.Vertical}
          />
          {children}
        </div>
      </div>
    </Page>
  );
};

export default PageMessage;
