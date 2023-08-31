import { Divider, Modal } from 'antd';
import React, { Fragment } from 'react';

import Json from 'components/Json';
import { PoolLogo } from 'components/ResourcePoolCard';
import css from 'components/ResourcePoolDetails.module.scss';
import { V1ResourcePoolDetail } from 'services/api-ts-sdk';
import { JsonObject, ResourcePool } from 'types';
import { camelCaseToSentence } from 'utils/string';

interface Props {
  finally?: () => void;
  resourcePool: ResourcePool;
  visible: boolean;
}

const ResourcePoolDetails: React.FC<Props> = ({ resourcePool: pool, ...props }: Props) => {
  const { details, ...mainSection } = pool;

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
      onCancel={props.finally}
      onOk={props.finally}>
      <Json json={mainSection as unknown as JsonObject} translateLabel={camelCaseToSentence} />
      {Object.keys(details).map((key) => (
        <Fragment key={key}>
          <Divider />
          <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
          <Json
            json={details[key as keyof V1ResourcePoolDetail] as unknown as JsonObject}
            translateLabel={camelCaseToSentence}
          />
        </Fragment>
      ))}
    </Modal>
  );
};

export default ResourcePoolDetails;
