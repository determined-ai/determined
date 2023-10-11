import { useObservable } from 'micro-observables';
import React from 'react';

import Logo, { Orientation } from 'components/Logo';
import Page from 'components/Page';
import determinedStore, { BrandingType } from 'stores/determinedInfo';

import Message from './kit/Message';

interface Props {
  children: React.ReactNode;
  title: string;
}

const PageMessage: React.FC<Props> = ({ title, children }: Props) => {
  const info = useObservable(determinedStore.info);
  return (
    <Page breadcrumb={[]} docTitle={title} noScroll>
      <Message
        description={children}
        icon={
          <Logo
            branding={info.branding || BrandingType.Determined}
            orientation={Orientation.Vertical}
          />
        }
        title={title}
      />
    </Page>
  );
};

export default PageMessage;
