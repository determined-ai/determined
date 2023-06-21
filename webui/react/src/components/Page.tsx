import { useObservable } from 'micro-observables';
import React, { MutableRefObject } from 'react';
import { Helmet } from 'react-helmet-async';

import { MenuItem } from 'components/kit/Dropdown';
import PageHeader from 'components/PageHeader';
import PageNotFound from 'components/PageNotFound';
import Spinner from 'components/Spinner';
import usePermissions from 'hooks/usePermissions';
import determinedStore, { BrandingType } from 'stores/determinedInfo';

import css from './Page.module.scss';

export interface BreadCrumbRoute {
  breadcrumbName: string;
  breadcrumbTooltip?: string;
  path: string;
}

export interface Props {
  bodyNoPadding?: boolean;
  breadcrumb: BreadCrumbRoute[];
  children?: React.ReactNode;
  containerRef?: MutableRefObject<HTMLElement | null>;
  docTitle?: string;
  headerComponent?: React.ReactNode;
  id?: string;
  ignorePermissions?: boolean;
  loading?: boolean;
  noScroll?: boolean;
  notFound?: boolean;
  onClickMenu?: (key: string) => void;
  options?: React.ReactNode;
  stickyHeader?: boolean;
  title?: string;
  menuItems?: MenuItem[];
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

  const classes = [css.base];

  if (props.bodyNoPadding) classes.push(css.bodyNoPadding);
  if (props.stickyHeader) classes.push(css.stickyHeader);
  if (props.noScroll) classes.push(css.noScroll);

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
        <article className={classes.join(' ')} id={props.id} ref={props.containerRef}>
          <PageHeader
            breadcrumb={props.breadcrumb}
            menuItems={props.menuItems}
            options={props.options}
            sticky={props.stickyHeader}
            onClickMenu={props.onClickMenu}
          />
          {props.headerComponent}
          <div className={css.body}>
            <Spinner spinning={!!props.loading}>{props.children}</Spinner>
          </div>
        </article>
      )}
    </>
  );
};

export default Page;
