import { useObservable } from 'micro-observables';
import React from 'react';

import Message from 'components/kit/Message';
import Logo from 'components/Logo';
import Page from 'components/Page';
import determinedStore from 'stores/determinedInfo';

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
        icon={<Logo branding={info.branding} orientation="vertical" />}
        title={title}
      />
    </Page>
  );
};

export default PageMessage;
