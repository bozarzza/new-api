import React, { useState, useEffect } from 'react';
import { Typography, Card, Space, Button, Table, Modal, Input, Tag } from '@douyinfe/semi-ui';
import { Coins, HandCoins, ExternalLink } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const Credit = () => {
  const { t } = useTranslation();
  const [balance, setBalance] = useState(0);
  const [loading, setLoading] = useState(false);
  const [transactions, setTransactions] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [withdrawModal, setWithdrawModal] = useState(false);
  const [withdrawAmount, setWithdrawAmount] = useState(0);

  const fetchBalance = async () => {
    try {
      const res = await API.get('/api/credit/balance');
      if (res.data.success) {
        setBalance(res.data.data.balance || 0);
      }
    } catch (e) {
      // ignore silently
    }
  };

  const fetchTransactions = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/credit/transactions', {
        params: { p: page, page_size: pageSize },
      });
      if (res.data.success) {
        setTransactions(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBalance();
  }, []);

  useEffect(() => {
    fetchTransactions();
  }, [page, pageSize]);

  const handleWithdraw = async () => {
    if (withdrawAmount <= 0) {
      showError(t('提现金额须大于0'));
      return;
    }
    try {
      const res = await API.post('/api/credit/withdraw', { amount: Number(withdrawAmount) });
      if (res.data.success) {
        showSuccess(t('提现请求已提交'));
        setWithdrawModal(false);
        fetchBalance();
        fetchTransactions();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const formatType = (type) => {
    switch (type) {
      case 'earn': return <Tag color="green">{t('收益')}</Tag>;
      case 'fee': return <Tag color="red">{t('佣金')}</Tag>;
      case 'withdraw': return <Tag color="orange">{t('提现')}</Tag>;
      case 'bonus': return <Tag color="blue">{t('奖励')}</Tag>;
      default: return <Tag color="grey">{type}</Tag>;
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { 
      title: t('时间'), 
      dataIndex: 'created_time',
      render: (v) => new Date(v * 1000).toLocaleString() 
    },
    { 
      title: t('类型'), 
      dataIndex: 'type',
      render: formatType
    },
    { 
      title: t('渠道ID'), 
      dataIndex: 'channel_id',
      render: (v) => v === 0 ? '-' : v
    },
    { 
      title: t('描述'), 
      dataIndex: 'description'
    },
    { 
      title: t('变动金额 ($)'), 
      dataIndex: 'amount',
      render: (v) => (
        <Text style={{ 
          color: v > 0 ? 'var(--semi-color-success)' : 'var(--semi-color-danger)', 
          fontWeight: 'bold' 
        }}>
          {v > 0 ? '+' : ''}{(v || 0).toFixed(5)}
        </Text>
      )
    },
    { 
      title: t('变动后余额'), 
      dataIndex: 'balance_after',
      render: (v) => <Text>{(v || 0).toFixed(5)}</Text>
    }
  ];

  return (
    <div style={{ padding: '0 24px 24px 24px', maxWidth: 1200, margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24, padding: '24px 0', borderBottom: '1px solid var(--semi-color-border)' }}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Coins size={32} style={{ color: 'var(--semi-color-warning)', marginRight: 12 }} />
          <div>
            <Title heading={2}>{t('积分中心')}</Title>
            <Text type="tertiary">{t('查看您的收益余额与提现流水')}</Text>
          </div>
        </div>
      </div>

      <Card style={{ marginBottom: 24, borderRadius: 12, background: 'linear-gradient(135deg, #fef2f2 0%, #fffbeb 100%)', border: 'none' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Text type="secondary" size="large">{t('当前可提现余额')}</Text>
            <Title heading={1} style={{ color: '#d97706', marginTop: 8 }}>
              ${balance.toFixed(5)}
            </Title>
          </div>
          <Button 
            size="large" 
            theme="solid" 
            style={{ backgroundColor: '#f59e0b', color: '#fff', borderRadius: 8, padding: '0 24px' }}
            icon={<HandCoins size={18} />}
            onClick={() => setWithdrawModal(true)}
          >
            {t('申请提现')}
          </Button>
        </div>
      </Card>

      <Title heading={4} style={{ marginBottom: 16 }}>{t('账单流水')}</Title>
      
      <Card style={{ borderRadius: 12 }} bodyStyle={{ padding: 0 }}>
        <Table 
          columns={columns} 
          dataSource={transactions} 
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
        title={t('申请提现')}
        visible={withdrawModal}
        onCancel={() => setWithdrawModal(false)}
        onOk={handleWithdraw}
      >
        <div style={{ padding: '16px 0' }}>
          <Text style={{ display: 'block', marginBottom: 8 }}>{t('提现金额 ($)')} (最高: {balance.toFixed(2)})</Text>
          <Input 
            type="number"
            value={withdrawAmount} 
            onChange={v => setWithdrawAmount(v)} 
            placeholder={t('请输入提现金额')}
            size="large"
          />
          <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 12 }}>
            * {t('提交申请后，管理员将在 1-3 个工作日内处理您的结算。详情请联系客服。')}
          </Text>
        </div>
      </Modal>
    </div>
  );
};

export default Credit;
