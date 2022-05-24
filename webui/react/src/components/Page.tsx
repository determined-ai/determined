import React from 'react';
import { Helmet } from 'react-helmet-async';

import { useStore } from 'contexts/Store';
import BasePage, { Props as BasePageProps } from 'tmp-shared/components/Page';
import { BrandingType } from 'types';

export interface Props extends BasePageProps {
  docTitle?: string;
}

const getFullDocTitle = (branding: string, title?: string, clusterName?: string) => {
  const brand = branding === BrandingType.HPE ?
    'HPE Machine Learning Development Environment' : 'Determined';
  const segmentList = [ brand ];

  if (clusterName) segmentList.unshift(clusterName);
  if (title) segmentList.unshift(title);

  return segmentList.join(' - ');
};

const Page: React.FC<Props> = (props: Props) => {
  const { info } = useStore();
  const brandingPath = `${process.env.PUBLIC_URL}/${info.branding}`;

  const docTitle = getFullDocTitle(
    info.branding,
    props.docTitle || props.title,
    info.clusterName,
  );

  return (
    <BasePage {...props}>
      <Helmet>
        <title>{docTitle}</title>
        {info.checked && (
          <>
            <link
              href={`${brandingPath}/favicon.ico`}
              rel="shortcut icon"
              type="image/x-icon"
            />
            <link href={`${brandingPath}/apple-touch-icon.png`} rel="apple-touch-icon" />
            <link href={`${brandingPath}/manifest.json`} rel="manifest" />
          </>
        )}
      </Helmet>
    </BasePage>
  );
};

export default Page;
