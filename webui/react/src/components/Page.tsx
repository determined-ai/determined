import React from 'react';
import { Helmet } from 'react-helmet-async';

import PageHeader from 'components/PageHeader';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import BasePage, { Props as BasePageProps } from 'shared/components/Page';
import Spinner from 'shared/components/Spinner';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { BrandingType } from 'types';
import { Loadable } from 'utils/loadable';

export interface Props extends Omit<BasePageProps, 'pageHeader'> {
  docTitle?: string;
  ignorePermissions?: boolean;
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
  const rbacEnabled = useFeature().isOn('rbac');
  const { loading: loadingPermissions } = usePermissions();

  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
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
      {!props.ignorePermissions && rbacEnabled && loadingPermissions ? (
        <Spinner center />
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
