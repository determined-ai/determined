import { InboxOutlined, UploadOutlined } from '@ant-design/icons';
import { ComponentStory, Meta } from '@storybook/react';
import {
  Button,
  Checkbox,
  Col,
  Form,
  Input,
  InputNumber,
  Radio,
  Rate,
  Row,
  Select,
  Slider,
  Switch,
  Upload,
} from 'antd';
import React from 'react';

const { Option } = Select;

export default {
  argTypes: {
    layout: { control: { options: ['vertical', 'horizontal', 'inline'], type: 'inline-radio' } },
    requiredMark: { control: { options: [true, false, 'optional'], type: 'inline-radio' } },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
  },
  component: Form,
  title: 'Ant Design/Form',
} as Meta<typeof Form>;

const normFile = (e: React.ChangeEvent<HTMLInputElement>) => {
  if (Array.isArray(e)) {
    return e;
  }
  return e?.target.files;
};

export const Default: ComponentStory<typeof Form> = (args) => {
  return (
    <Form {...args}>
      <Form.Item
        label="Username"
        name="username"
        rules={[{ message: 'Please input your username!', required: true }]}>
        <Input />
      </Form.Item>
      <Form.Item
        label="Password"
        name="password"
        rules={[{ message: 'Please input your password!', required: true }]}>
        <Input.Password />
      </Form.Item>
      <Form.Item name="remember" valuePropName="checked">
        <Checkbox>Remember me</Checkbox>
      </Form.Item>
      <Form.Item label="Plain Text">
        <span className="ant-form-text">China</span>
      </Form.Item>
      <Form.Item
        hasFeedback
        label="Select"
        name="select"
        rules={[{ message: 'Please select your country!', required: true }]}>
        <Select placeholder="Please select a country">
          <Option value="china">China</Option>
          <Option value="usa">U.S.A</Option>
        </Select>
      </Form.Item>
      <Form.Item
        label="Select[multiple]"
        name="select-multiple"
        rules={[
          {
            message: 'Please select your favourite colors!',
            required: true,
            type: 'array',
          },
        ]}>
        <Select mode="multiple" placeholder="Please select favourite colors">
          <Option value="red">Red</Option>
          <Option value="green">Green</Option>
          <Option value="blue">Blue</Option>
        </Select>
      </Form.Item>
      <Form.Item label="InputNumber">
        <Form.Item name="input-number" noStyle>
          <InputNumber max={10} min={1} />
        </Form.Item>
        <span className="ant-form-text"> machines</span>
      </Form.Item>
      <Form.Item label="Switch" name="switch" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Form.Item label="Slider" name="slider">
        <Slider
          marks={{
            0: 'A',
            20: 'B',
            40: 'C',
            60: 'D',
            80: 'E',
            100: 'F',
          }}
        />
      </Form.Item>
      <Form.Item label="Radio.Group" name="radio-group">
        <Radio.Group>
          <Radio value="a">item 1</Radio>
          <Radio value="b">item 2</Radio>
          <Radio value="c">item 3</Radio>
        </Radio.Group>
      </Form.Item>
      <Form.Item
        label="Radio.Button"
        name="radio-button"
        rules={[{ message: 'Please pick an item!', required: true }]}>
        <Radio.Group>
          <Radio.Button value="a">item 1</Radio.Button>
          <Radio.Button value="b">item 2</Radio.Button>
          <Radio.Button value="c">item 3</Radio.Button>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="Checkbox.Group" name="checkbox-group">
        <Checkbox.Group>
          <Row>
            <Col span={8}>
              <Checkbox style={{ lineHeight: '32px' }} value="A">
                A
              </Checkbox>
            </Col>
            <Col span={8}>
              <Checkbox disabled style={{ lineHeight: '32px' }} value="B">
                B
              </Checkbox>
            </Col>
            <Col span={8}>
              <Checkbox style={{ lineHeight: '32px' }} value="C">
                C
              </Checkbox>
            </Col>
            <Col span={8}>
              <Checkbox style={{ lineHeight: '32px' }} value="D">
                D
              </Checkbox>
            </Col>
            <Col span={8}>
              <Checkbox style={{ lineHeight: '32px' }} value="E">
                E
              </Checkbox>
            </Col>
            <Col span={8}>
              <Checkbox style={{ lineHeight: '32px' }} value="F">
                F
              </Checkbox>
            </Col>
          </Row>
        </Checkbox.Group>
      </Form.Item>
      <Form.Item label="Rate" name="rate">
        <Rate />
      </Form.Item>
      <Form.Item
        extra="longgggggggggggggggggggggggggggggggggg"
        getValueFromEvent={normFile}
        label="Upload"
        name="upload"
        valuePropName="fileList">
        <Upload action="/upload.do" listType="picture" name="logo">
          <Button icon={<UploadOutlined />}>Click to upload</Button>
        </Upload>
      </Form.Item>
      <Form.Item label="Dragger">
        <Form.Item getValueFromEvent={normFile} name="dragger" noStyle valuePropName="fileList">
          <Upload.Dragger action="/upload.do" name="files">
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">Click or drag file to this area to upload</p>
            <p className="ant-upload-hint">Support for a single or bulk upload.</p>
          </Upload.Dragger>
        </Form.Item>
      </Form.Item>
      <Form.Item wrapperCol={{ offset: 6, span: 12 }}>
        <Button htmlType="submit" type="primary">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

Default.args = { layout: 'vertical', requiredMark: true, size: 'middle' };
