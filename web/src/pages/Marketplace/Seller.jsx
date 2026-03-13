import React, { useState, useEffect } from 'react';
import { Typography, Card, Space, Button, Table, Tag, Popconfirm, Modal, Form, Input, InputNumber, Select } from '@douyinfe/semi-ui';
import { ShoppingBag, Plus, Edit, Trash } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const Seller = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [channels, setChannels] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [modalVisible, setModalVisible] = useState(false);
  const [editingId, setEditingId] = useState(null);
  
  const [form] = Form.useForm();

  const fetchChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/marketplace/channels/self', {
        params: { p: page, page_size: pageSize },
      });
      if (res.data.success) {
        setChannels(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchChannels();
  }, [page, pageSize]);

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/marketplace/channels/${id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        fetchChannels();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const handleModalOpen = (record = null) => {
    setEditingId(record ? record.id : null);
    if (record) {
      form.setValues({
        name: record.name,
        models: record.models,
        price_per_k_token: record.price_per_k_token,
        channel_label: record.channel_label,
        daily_token_limit: record.daily_token_limit,
        max_concurrent: record.max_concurrent,
      });
    } else {
      form.reset();
      form.setValues({
        type: 1, // OpenAI
        price_per_k_token: 0.002,
        max_concurrent: 0,
        daily_token_limit: 0,
      });
    }
    setModalVisible(true);
  };

  const handleSubmit = async (values) => {
    try {
      if (editingId) {
        // Edit only specific fields
        const payload = {
          name: values.name,
          models: values.models,
          price_per_k_token: Number(values.price_per_k_token),
          channel_label: values.channel_label,
          daily_token_limit: Number(values.daily_token_limit || 0),
          max_concurrent: Number(values.max_concurrent || 0),
        };
        const res = await API.put(`/api/marketplace/channels/${editingId}`, payload);
        if (res.data.success) {
          showSuccess(t('更新成功'));
          setModalVisible(false);
          fetchChannels();
        } else {
          showError(res.data.message);
        }
      } else {
        // Add full channel
        const payload = {
          ...values,
          price_per_k_token: Number(values.price_per_k_token),
          daily_token_limit: Number(values.daily_token_limit || 0),
          max_concurrent: Number(values.max_concurrent || 0),
          type: Number(values.type || 1),
        };
        const res = await API.post('/api/marketplace/channels', payload);
        if (res.data.success) {
          showSuccess(t('上架成功'));
          setModalVisible(false);
          fetchChannels();
        } else {
          showError(res.data.message);
        }
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: t('名称'), dataIndex: 'name' },
    { 
      title: t('模型'), 
      dataIndex: 'models',
      render: (text) => (
        <div style={{ maxWidth: 200, WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', display: '-webkit-box', overflow: 'hidden' }}>
          {text}
        </div>
      )
    },
    { 
      title: t('价格/1k'), 
      dataIndex: 'price_per_k_token',
      render: (val) => <Text style={{ color: 'var(--semi-color-primary)', fontWeight: 'bold' }}>${val?.toFixed(3)}</Text>
    },
    { 
      title: t('评分'), 
      render: (_, r) => <Text>{r.avg_rating.toFixed(1)} ({r.rating_count})</Text> 
    },
    { 
      title: t('标签'), 
      dataIndex: 'channel_label',
      render: (v) => v ? <Tag color="blue">{v}</Tag> : '-'
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (v) => v === 1 ? <Tag color="green">{t('在线')}</Tag> : <Tag color="red">{t('离线')}</Tag>
    },
    {
      title: t('操作'),
      render: (_, record) => (
        <Space>
          <Button icon={<Edit size={14}/>} theme="light" onClick={() => handleModalOpen(record)} />
          <Popconfirm title={t('确定下架此渠道吗？')} onConfirm={() => handleDelete(record.id)}>
            <Button icon={<Trash size={14}/>} theme="light" type="danger" />
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div style={{ padding: '0 24px 24px 24px', maxWidth: 1400, margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24, padding: '24px 0', borderBottom: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <ShoppingBag size={32} style={{ color: 'var(--semi-color-primary)', marginRight: 12 }} />
          <div>
            <Title heading={2}>{t('我的上架')}</Title>
            <Text type="tertiary">{t('管理您在算力市场上架的 API 渠道')}</Text>
          </div>
        </div>
        <Button icon={<Plus size={16}/>} theme="solid" type="primary" onClick={() => handleModalOpen(null)}>
          {t('上架新渠道')}
        </Button>
      </div>

      <Card style={{ borderRadius: 12 }} bodyStyle={{ padding: 0 }}>
        <Table 
          columns={columns} 
          dataSource={channels} 
          rowKey="id"
          loading={loading}
          pagination={{
            currentPage: page,
            pageSize: pageSize,
            total,
            onPageChange: setPage,
            onPageSizeChange: setPageSize,
          }}
        />
      </Card>

      <Modal
        title={editingId ? t('编辑渠道') : t('上架新渠道')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form initValues={{}} getFormApi={f => {}} form={form} onSubmit={handleSubmit} labelPosition="top">
          <Form.Input field="name" label={t('渠道名称')} required />
          
          {!editingId && (
            <>
              <Form.Select field="type" label={t('渠道类型')} required style={{ width: '100%' }}>
                <Form.Select.Option value={1}>OpenAI Compatible</Form.Select.Option>
                <Form.Select.Option value={14}>Anthropic</Form.Select.Option>
                <Form.Select.Option value={24}>Gemini</Form.Select.Option>
              </Form.Select>
              <Form.Input field="base_url" label={t('代理地址 (Base URL)')} required />
              <Form.Input field="key" label={t('密钥 (Key)')} required />
            </>
          )}

          <Form.Input field="models" label={t('模型列表 (逗号分隔)')} required />
          <Form.InputNumber field="price_per_k_token" label={t('单价 ($ / 1K Token)')} step={0.001} required width="100%" />
          <Form.Input field="channel_label" label={t('渠道标签 (如 official)')} />
          
          <Space spacing="loose" style={{ width: '100%' }}>
             <Form.InputNumber field="daily_token_limit" label={t('日 Token 限制 (0表不限)')} width="100%" />
             <Form.InputNumber field="max_concurrent" label={t('并发限制 (0表不限)')} width="100%" />
          </Space>

          <Button theme="solid" type="primary" htmlType="submit" block style={{ marginTop: 24 }}>
            {editingId ? t('保存修改') : t('确认上架')}
          </Button>
        </Form>
      </Modal>
    </div>
  );
};

export default Seller;
