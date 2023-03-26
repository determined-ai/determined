import { useObservable } from 'micro-observables';
import React from 'react';
import { Helmet } from 'react-helmet-async';

import PageHeader from 'components/PageHeader';
import PageNotFound from 'components/PageNotFound';
import usePermissions from 'hooks/usePermissions';
import BasePage, { Props as BasePageProps } from 'shared/components/Page';
import Spinner from 'shared/components/Spinner';
import determinedStore, { BrandingType } from 'stores/determinedInfo';

export interface Props extends Omit<BasePageProps, 'pageHeader'> {
  docTitle?: string;
  ignorePermissions?: boolean;
  notFound?: boolean;
}

const getFullDocTitle = (branding: string, title?: string, clusterName?: string) => {
  const brand =
    branding === BrandingType.HPE ? 'HPE Machine Learning Development Environment' : 'Determined';
  const segmentList = [brand];

  if (clusterName) segmentList.unshift(clusterName);
  if (title) segmentList.unshift(title);

  return segmentList.join(' - ');
};

const Page: React.FC<Props> = (props: Props) => {
  const { loading: loadingPermissions } = usePermissions();

  const info = useObservable(determinedStore.info);
  const branding = info.branding || BrandingType.Determined;
  const brandingPath = `${process.env.PUBLIC_URL}/${branding}`;

  const docTitle = getFullDocTitle(branding, props.docTitle || props.title, info.clusterName);

  return (
    <>
      <Helmet>
        <title>{docTitle}</title>
        {info.checked && (
          <>
            <link href={`${brandingPath}/favicon.ico`} rel="shortcut icon" type="image/x-icon" />
            <link href={`${brandingPath}/apple-touch-icon.png`} rel="apple-touch-icon" />
            <link href={`${brandingPath}/manifest.json`} rel="manifest" />
          </>
        )}
      </Helmet>
      {!props.ignorePermissions && loadingPermissions ? (
        <Spinner center />
      ) : props.notFound ? (
        <PageNotFound /> // hide until permissions are loaded
      ) : (
        <BasePage
          {...props}
          pageHeader={
            <PageHeader
              breadcrumb={props.breadcrumb}
              options={props.options}
              sticky={props.stickyHeader}
              subTitle={props.subTitle}
              title={props.title}
            />
          }
        />
      )}
    </>
  );
};

export default Page;
