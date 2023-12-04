import { Divider, Modal } from 'antd';
import { useTheme } from 'hew/Theme';
import React, { Fragment } from 'react';

import JsonGlossary from 'components/JsonGlossary';
import { V1ResourcePoolDetail } from 'services/api-ts-sdk';
import { JsonObject, ResourcePool } from 'types';
import { camelCaseToSentence } from 'utils/string';

import { PoolLogo } from './ResourcePoolCard';
import css from './ResourcePoolDetails.module.scss';

interface Props {
  finally?: () => void;
  resourcePool: ResourcePool;
  visible: boolean;
}

const ResourcePoolDetails: React.FC<Props> = ({ resourcePool: pool, ...props }: Props) => {
  const { details, ...mainSection } = pool;

  const {
    themeSettings: { className: themeClass },
  } = useTheme();

  for (const key in details) {
    if (details[key as keyof V1ResourcePoolDetail] === null) {
      delete details[key as keyof V1ResourcePoolDetail];
    }
  }

  const title = (
    <div>
      <PoolLogo type={pool.type} />
      {' ' + pool.name}
    </div>
  );

  return (
    <Modal
      cancelButtonProps={{ style: { display: 'none' } }}
      cancelText=""
      mask
      open={props.visible}
      style={{ minWidth: '600px' }}
      title={title}
      wrapClassName={themeClass}
      onCancel={props.finally}
      onOk={props.finally}>
      <JsonGlossary
        json={mainSection as unknown as JsonObject}
        translateLabel={camelCaseToSentence}
      />
      {Object.keys(details).map((key) => (
        <Fragment key={key}>
          <Divider />
          <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
          <JsonGlossary
            json={details[key as keyof V1ResourcePoolDetail] as unknown as JsonObject}
            translateLabel={camelCaseToSentence}
          />
        </Fragment>
      ))}
    </Modal>
  );
};

export default ResourcePoolDetails;
