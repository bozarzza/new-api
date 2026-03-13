import React, { useState, useEffect } from 'react';
import { Typography, Card, Space, Input, Button, Tag, Pagination, Rating, Popover, Spin, Select, Form } from '@douyinfe/semi-ui';
import { Search, Store, Zap, ThumbsUp, MapPin } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { renderModelTag, stringToColor } from '../../helpers/render';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const Marketplace = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [channels, setChannels] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(12);

  const [filterModel, setFilterModel] = useState('');
  const [filterLabel, setFilterLabel] = useState('');
  const [priceMax, setPriceMax] = useState('');

  const [ratingVisible, setRatingVisible] = useState(false);
  const [currentChannel, setCurrentChannel] = useState(null);
  const [ratingValue, setRatingValue] = useState(5);
  const [ratingComment, setRatingComment] = useState('');

  const fetchChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/marketplace/channels', {
        params: {
          p: page,
          page_size: pageSize,
          model: filterModel,
          label: filterLabel,
          max_price: priceMax,
        },
      });
      if (res.data.success) {
        setChannels(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchChannels();
  }, [page, pageSize]);

  const handleSearch = () => {
    setPage(1);
    fetchChannels();
  };

  const submitRating = async () => {
    if (!currentChannel) return;
    try {
      const res = await API.post(`/api/marketplace/channels/${currentChannel.id}/rate`, {
        score: ratingValue,
        comment: ratingComment,
      });
      if (res.data.success) {
        showSuccess(t('评价成功'));
        setRatingVisible(false);
        fetchChannels();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div style={{ padding: '0 24px 24px 24px', maxWidth: 1400, margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 24, padding: '24px 0', borderBottom: '1px solid var(--semi-color-border)' }}>
        <Store size={32} style={{ color: 'var(--semi-color-primary)', marginRight: 12 }} />
        <div>
          <Title heading={2}>{t('算力市场 (Marketplace)')}</Title>
          <Text type="tertiary">{t('发现高质量、高性价比的第三方模型 API 渠道')}</Text>
        </div>
      </div>

      <Card style={{ marginBottom: 24, borderRadius: 12 }}>
        <Space wrap spacing="loose">
          <Input 
            prefix={<Search size={16} />} 
            placeholder={t('搜索模型，如 gpt-4')} 
            value={filterModel}
            onChange={setFilterModel}
            onEnterPress={handleSearch}
          />
          <Input 
            placeholder={t('渠道标签，如 official')} 
            value={filterLabel}
            onChange={setFilterLabel}
            onEnterPress={handleSearch}
          />
          <Input 
            placeholder={t('最大价格 $')} 
            type="number"
            value={priceMax}
            onChange={setPriceMax}
            onEnterPress={handleSearch}
          />
          <Button theme="solid" type="primary" onClick={handleSearch}>{t('筛选')}</Button>
        </Space>
      </Card>

      <Spin spinning={loading}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 24, paddingBottom: 24 }}>
          {channels.map((ch) => (
            <Card 
              key={ch.id} 
              shadows="hover"
              style={{ borderRadius: 16, overflow: 'hidden', border: '1px solid var(--semi-color-border)' }}
              bodyStyle={{ padding: 24 }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
                <div>
                  <Typography.Title heading={5} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {ch.name}
                    {ch.status === 1 ? <Tag color="green" size="small">在线</Tag> : <Tag color="red" size="small">离线</Tag>}
                  </Typography.Title>
                  <Text type="tertiary" size="small">ID: {ch.id}</Text>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <Text style={{ fontSize: 20, fontWeight: 'bold', color: 'var(--semi-color-primary)' }}>
                    ${ch.price_per_k_token.toFixed(3)}
                  </Text>
                  <Text type="tertiary" size="small" style={{ display: 'block' }}>/ 1k Token</Text>
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <Text type="tertiary" size="small">{t('模型支持')}: </Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
                  {ch.models.split(',').slice(0, 5).map(m => (
                    <Tag key={m} color={stringToColor(m)} size="small">{m}</Tag>
                  ))}
                  {ch.models.split(',').length > 5 && <Tag size="small" type="ghost">+{ch.models.split(',').length - 5}</Tag>}
                </div>
              </div>

              <Space spacing="loose" style={{ marginBottom: 16, width: '100%' }}>
                <Text style={{ display: 'flex', alignItems: 'center' }} type="secondary" size="small">
                  <Zap size={14} style={{ marginRight: 4, color: '#f5a623' }} /> {ch.response_time}ms
                </Text>
                {ch.channel_label && (
                  <Text style={{ display: 'flex', alignItems: 'center' }} type="secondary" size="small">
                    <MapPin size={14} style={{ marginRight: 4, color: '#4a90e2' }} /> {ch.channel_label}
                  </Text>
                )}
              </Space>

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderTop: '1px solid var(--semi-color-border)', paddingTop: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Rating allowHalf value={ch.avg_rating} size="small" disabled />
                  <Text size="small" type="tertiary">({ch.rating_count})</Text>
                </div>
                <Button 
                  icon={<ThumbsUp size={14} />} 
                  theme="light" 
                  size="small"
                  onClick={() => {
                    setCurrentChannel(ch);
                    setRatingValue(5);
                    setRatingComment('');
                    setRatingVisible(true);
                  }}
                >
                  {t('评价')}
                </Button>
              </div>
            </Card>
          ))}
        </div>
        
        {channels.length === 0 && !loading && (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <Text type="tertiary">{t('暂无符合条件的渠道')}</Text>
          </div>
        )}
      </Spin>

      {total > 0 && (
        <div style={{ display: 'flex', justifyContent: 'center' }}>
          <Pagination 
            total={total} 
            currentPage={page} 
            pageSize={pageSize} 
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
            showSizeChanger
          />
        </div>
      )}

      <Modal
        title={`${t('评价渠道')} ${currentChannel?.name}`}
        visible={ratingVisible}
        onCancel={() => setRatingVisible(false)}
        footer={
           <Button theme="solid" type="primary" onClick={submitRating}>
              {t('提交评价')}
           </Button>
        }
      >
         <div style={{ marginBottom: 16 }}>
            <Text>{t('评分')}</Text>
            <div style={{ marginTop: 8 }}>
              <Rating value={ratingValue} onChange={setRatingValue} allowHalf />
            </div>
         </div>
         <div style={{ marginBottom: 24 }}>
            <Text>{t('评论 (可选)')}</Text>
            <Input.TextArea 
              style={{ marginTop: 8 }} 
              value={ratingComment} 
              onChange={setRatingComment} 
              rows={3} 
              placeholder={t('您的使用体验如何？')}
            />
         </div>
      </Modal>
    </div>
  );
};

export default Marketplace;
