# OCTEON PCI 虚拟网卡原理分析

## 概述

---

OCTEON CPU提供对PCI/PCIE总线的支持，其中，它既可以当做EP(Pcie endpoint，俗称从设备)也可以当做RC(Root complex，俗称主设备),在本文中，OCTEON充当的是EP，也就是说，octeon完全是由pcie总线上的rc控制，在ARES的硬件设计中，由于cavium和x86是直连的分布，所以rc就是x86。

在cavium上提供两个PCIE Mac ports，以提供对pcie的支持,
通过这两个ports，外部可以访问cavium自身的memory，cavium也可以通过它访问外部的内存,从而实现ep和rc之间的数据交互。
下面是关于这部分的一个框图：

![sli_diag](/public/images/oct_pcie/sli_diagram.png)

cavium将pci可以访问的资源做了一个抽象，其中主要有下面几种：

- 用于收发的数据包，它们分别被放在输入输出队列中。
- 专门用于dma传输的单元。
- 外部可以映射的pcie bar空间。

基于上述的几种资源，cavium使用一个叫做的sli(Switch Logic Interface)的协处理器，处理所有的资源访问请求。
下面将结合rc端的驱动，讲解数据包的收发过程(其中发送和接收都是相对于rc而言,
中间为了防止造成干扰，将忽略部分内容，统一放到后续章节具体介绍)。

## 发送

---

### 基本原理

cavium通过一个输入队列来接收rc的数据包，队列中的每个entry的结构如下图：

![in_entry](/public/images/oct_pcie/in_entry.png)

- DPTR:指向数据部分的指针。
- DPI_INST_HDR:每个entry的head，用于描述该数据的属性，具体的字段如下：

![entry_head](/public/images/oct_pcie/entry_head.png)

主要包括数据包的group，length，tag，qos等等。
其中有个gather list的概念，它的意思是该entry中的DPTR指向的不是数据的内容，而是指向一系列数据的指针，是一个二级指针。
这样便可以通过一个entry来发送多个数据包，这里我们没有用到该技术。

当sli收到一个这样的entry便会把它放到一个专门的input queue中，
随后ipd从这个队列中获取到entry，并根据entry中指针，将远端数据拷贝到本地。
最后根据entry head中的属性，形成一个work entry上送到sso，
最终core收到该数据包，完成这个数据发送的过程。
这样在core看来，这和正常从xaui口收上来的包别无二样。

### 关键代码分析

---

#### 初始化输入队列

对于数据队列的初始化，主要是申请dma内存，并以此初始化sli的相关寄存器。

首先是初始化管理数据结构:

~~~
octeon_setup_instr_queues:

	for(i = 0; i < num_iqs; i++) {
		oct->instr_queue[i] = cavium_alloc_virt(sizeof(octeon_instr_queue_t));
		if(oct->instr_queue[i] == NULL)
			return 1;

		cavium_memset(oct->instr_queue[i], 0, sizeof(octeon_instr_queue_t));

		if(octeon_init_instr_queue(oct, i))
			return 1;

		oct->num_iqs++;
	}
~~~

接着是申请用于数据发送的dma内存：

~~~
octeon_setup_instr_queues
 ->octeon_init_instr_queue:

	q_size = conf->instr_type * conf->num_descs;

	iq = oct->instr_queue[iq_no];

	iq->base_addr = octeon_pci_alloc_consistent(oct->pci_dev, q_size, &iq->base_addr_dma);
	if(!iq->base_addr) {
		cavium_error("OCTEON: Cannot allocate memory for instr queue %d\n",iq_no);
		return 1;
	}
~~~

最后初始化sli相关寄存器, 主要将物理地址，队列的长度，以及队列的id告诉cavium:

~~~
octeon_setup_instr_queues
 ->octeon_init_instr_queue
  ->cn68xx_setup_iq_regs:

	/* Write the start of the input queue's ring and its size  */
	octeon_write_csr64(oct,CN68XX_SLI_IQ_BASE_ADDR64(iq_no),iq->base_addr_dma);
	octeon_write_csr(oct, CN68XX_SLI_IQ_SIZE(iq_no), iq->max_count);

	octeon_write_csr(oct, CN68XX_SLI_IQ_PORT_PKIND(iq_no), iq_no);
~~~

#### 发送一个数据包

当从协议栈收到一个数据包时，最先调用是driver注册的发送回调函数`octnet_xmit`,
该函数的主要是根据从协议栈收到的数据包形成一个input entry

~~~
octnet_xmit:

	/* Prepare the attributes for the data to be passed to OSI. */
	...
	ndata.q_no      = priv->txq;
	ndata.datasize  = skb->len;

	cmdsetup.u64       = 0;
	cmdsetup.s.ifidx = priv->linfo.ifidx;

	...

	cmdsetup.s.u.datasize = skb->len;

	octnet_prepare_pci_cmd(&(ndata.cmd), &cmdsetup);

	ndata.cmd.dptr =
			  octeon_map_single_buffer(get_octeon_device_id(priv->oct_dev),
							  skb->data, skb->len, CAVIUM_PCI_DMA_TODEVICE);
	ndata.buftype   = NORESP_BUFTYPE_NET;

	...

	status = octnet_send_nic_data_pkt(priv->oct_dev, &ndata);
	if(status == NORESP_SEND_FAILED)
		goto oct_xmit_failed;
~~~
这里主要是设置数据包的长度，数据的发送到的queue的id(一个port最多支持32个queue)。
最后将设置好的entry通知cavium，让其进行处理。

首先看一下entry是如何形成的：

~~~
octnet_xmit
 ->octnet_prepare_pci_cmd:
   
	ih           = (octeon_instr_ih_t *)&cmd->ih;

	ih->tagtype  = ORDERED_TAG;
	ih->grp      = OCTNET_POW_GRP;
	ih->tag      = 0x11111111 + setup->s.ifidx;
	ih->raw      = 1;


	if(!setup->s.gather) {
		ih->dlengsz  = setup->s.u.datasize;
	} else {
		ih->gather   = 1;
		ih->dlengsz  = setup->s.u.gatherptrs;
	}
~~~
- tagtype:order mode)
- group为:(这里需要在d-plane中有相应的group与之对应)
- dlengsz:如果只有一个数据包，则为该数据包的长度。
否则，则使用gather list模式(这里主要用于处理分片报文)

最后我们来看一下通知cavium的方式:

~~~
octnet_xmit
 ->octnet_send_nic_data_pkt
  ->octeon_send_noresponse_command
   ->__post_command2:

   /* This ensures that the read index does not wrap around to the same 
      position if queue gets full before Octeon could fetch any instr. */
   if(cavium_atomic_read(&iq->instr_pending) >= (iq->max_count - 1)) {
      cavium_print(PRINT_FLOW, "OCTEON[%d]: IQ[%d] is full (%d entries)\n",
                    octeon_dev->octeon_id, iq->iq_no,
                    cavium_atomic_read(&iq->instr_pending));
      cavium_print(PRINT_FLOW, "%s write_idx: %d flush: %d new: %d\n", __CVM_FUNCTION__,
                   iq->host_write_index, iq->flush_index,iq->octeon_read_index);

      st.status = IQ_SEND_FAILED; st.index = -1;
      return st;
   }

   if(cavium_atomic_read(&iq->instr_pending) >= (iq->max_count - 2))
      st.status = IQ_SEND_STOP;

   __copy_cmd_into_iq(iq, cmd);


   /* "index" is returned, host_write_index is modified. */
   st.index = iq->host_write_index;
   INCR_INDEX_BY1(iq->host_write_index, iq->max_count);
   iq->fill_cnt++;

   /* Flush the command into memory. */
   cavium_flush_write();

   if(iq->fill_cnt >= iq->fill_threshold || force_db)
         ring_doorbell(iq);

   cavium_atomic_inc(&iq->instr_pending);

octnet_xmit
 ->octnet_send_nic_data_pkt
  ->octeon_send_noresponse_command
   ->__post_command2
    ->ring_doorbell:

   OCTEON_WRITE32(iq->doorbell_reg, iq->fill_cnt);
   iq->fill_cnt     = 0;
   iq->last_db_time = cavium_jiffies;
   return ;
~~~
可见，最终的通知方式是用的doorbell。
这里有几个关于统计的值，需要先说下

- `instr_pending`: 已经发送，但是cavium未处理的entry的个数
- `host_write_index`: 指向下一个可用的空间。每发送一个entry，该值+1，如果达到队列的capacity，反转，从而实现ring queue。
- `fill_cnt`: 一次doorbell需要处理的entry的个数。每发送一个entry，该值+1，每进行一次doorbell，清0。

明白了这几个统计值，上面的代码不难理解。
如果队列中没有空间存放需要发送的entry，发送失败。
如果队列中只有一个空间，继续发送，但是停止后续的发送，直到有更多的空间。
如果是正常发送，将entry拷贝到input queue中的相应位置

~~~
octnet_xmit
 ->octnet_send_nic_data_pkt
  ->octeon_send_noresponse_command
   ->__post_command2
    ->__copy_cmd_into_iq:

	cmdsize = ((iq->iqcmd_64B)?64:32);
	iqptr = iq->base_addr + (cmdsize * iq->host_write_index);

	cavium_memcpy(iqptr, cmd, cmdsize);
~~~

最后根据发送的结果，更新相应的统计计数:

~~~
octnet_xmit
 ->octnet_send_nic_data_pkt
  ->octeon_send_noresponse_command:

	if(st.status != IQ_SEND_FAILED) {
		...
		INCR_INSTRQUEUE_PKT_COUNT(oct, iq_no, bytes_sent, datasize);
		INCR_INSTRQUEUE_PKT_COUNT(oct, iq_no, instr_posted, 1);
	} else {
		INCR_INSTRQUEUE_PKT_COUNT(oct, iq_no, instr_dropped, 1);
	}
~~~

这样整个的发送流程便结束了，下面将介绍接收流程。

## 接收

---

### 基本原理

数据包的接收通过中断来触发的, 这里有几个问题需要回答：

Q:那么cavium是根据什么条件来触发中断的呢？

A:这里有两个标准：数据包的个数和时间间隔。可以根据具体需求和环境进行不同的配置。

Q:那么cavium将数据包发到哪里呢？

A:rc会提供的output queue，和input queue类似，队列中的entry包含2个指针：

![out_entry](/public/images/oct_pcie/out_entry.png)

其中`buffer pointer`指向数据包的内容，
而`info pointer`则指向一个描述数据包属性的结构体：

![info_entry](/public/images/oct_pcie/info_entry.png)

好了，明确了这两个问题，那么接收的流程大致是这样的：

cavium将数据包的内容拷贝到`buffer pointer`所指向的远端内存中，
然后，将数据的长度，grp，qos等等一些属性拷贝到`info pointer`所指向的远端内存中，
最后，根据中断触发的条件向rc触发一个msi中断。
在接收端的中断回调函数里，将数据包上送至协议栈进行后续的协议部分的处理。

### 关键代码分析

---

#### 初始化输出队列

和输入队列初始化类似，这里的初始化工作是申请dma内存，并初始化sli寄存器。

首先是初始化管理数据结构：

~~~
octeon_setup_output_queues:

	oct->num_oqs = 0;

	for(i = 0; i < num_oqs; i++) {
		oct->droq[i] = cavium_alloc_virt(sizeof(octeon_droq_t));
		if(oct->droq[i] == NULL)
			return 1;

		cavium_memset(oct->droq[i], 0, sizeof(octeon_droq_t));

		if(octeon_init_droq(oct, i))
			return 1;

		oct->num_oqs++;
	}

	return 0;
~~~

关于申请dma内存, 这里分为3部分:

- output queue entry
- info entry
- buffer entry

首先是output queue entry:

~~~
octeon_setup_output_queues
 ->octeon_init_droq:

	desc_ring_size  = droq->max_count * OCT_DROQ_DESC_SIZE;
	droq->desc_ring = octeon_pci_alloc_consistent(oct->pci_dev, desc_ring_size,
	                                              &droq->desc_ring_dma);

	if(!droq->desc_ring)  {
		cavium_error("OCTEON: Output queue %d ring alloc failed\n", q_no);
		return 1;
	}
~~~
**NOTE:**这里分配的是连续的dma内存(即一致性dma)

关于`info entry`和`buffer entry`采用的是另一种dma申请方式:
即先申请内存，然后再建立dma映射(即流式dma)

首先是内存申请：

~~~
octeon_setup_output_queues
 ->octeon_init_droq:

	droq->info_list =
	cavium_alloc_aligned_memory((droq->max_count * OCT_DROQ_INFO_SIZE),
	                            &droq->info_alloc_size, &droq->info_base_addr);

	if(!droq->info_list) {
		cavium_error("OCTEON: Cannot allocate memory for info list.\n");
		octeon_pci_free_consistent(oct->pci_dev,
		                           (droq->max_count * OCT_DROQ_DESC_SIZE),
		                           droq->desc_ring,
		                           droq->desc_ring_dma);
		return 1;
	}
	cavium_print(PRINT_DEBUG,"setup_droq: droq_info: 0x%p\n", droq->info_list);

	droq->recv_buf_list = (octeon_recv_buffer_t *)
	                cavium_alloc_virt(droq->max_count * OCT_DROQ_RECVBUF_SIZE);
	if(!droq->recv_buf_list)  {
		cavium_error("OCTEON: Output queue recv buf list alloc failed\n");
		goto init_droq_fail;
	}
~~~
**NOTE:**`info pointer`必须是8字节对齐。

接着是建立流式dma映射：

~~~
octeon_setup_output_queues
 ->octeon_init_droq
  ->octeon_droq_setup_ring_buffers:

	...
	for(i = 0; i < droq->max_count; i++)  {
		...
		droq->info_list[i].length = 0;

		desc_ring[i].info_ptr = 
			(uint64_t)octeon_pci_map_single(oct->pci_dev, &droq->info_list[i], 
			OCT_DROQ_INFO_SIZE, CAVIUM_PCI_DMA_FROMDEVICE);

		...
		desc_ring[i].buffer_ptr = 
			(uint64_t)octeon_pci_map_single(oct->pci_dev, droq->recv_buf_list[i].data, 
			droq->buffer_size, CAVIUM_PCI_DMA_FROMDEVICE );
   }
   ...
~~~

分配好了dma内存，接着便是告诉cavium，即配置sli相应的寄存器：

~~~
octeon_setup_output_queues
 ->octeon_init_droq
  ->cn68xx_setup_oq_regs:

	...
	octeon_write_csr64(oct, CN68XX_SLI_OQ_BASE_ADDR64(oq_no), droq->desc_ring_dma);
	octeon_write_csr(oct, CN68XX_SLI_OQ_SIZE(oq_no), droq->max_count);

	octeon_write_csr(oct, CN68XX_SLI_OQ_BUFF_INFO_SIZE(oq_no),
	        (droq->buffer_size | (OCT_RESP_HDR_SIZE << 16)));
	...
~~~

#### 接收一个数据包

下面针对数据包的接收，梳理一下整个流程。
数据包的接收首先从中断回调函数讲起

~~~
octeon_setup_interrupt:

	irqret = request_irq(oct->pci_dev->irq, octeon_intr_handler,
		                 CVM_SHARED_INTR, "octeon", oct);
octeon_intr_handler
 ->cn68xx_interrupt_handler:
	intr64 = OCTEON_READ64(cn68xx->intr_sum_reg64);

	/* If our device has interrupted, then proceed. */
	if (!intr64)
		return CVM_INTR_NONE;

	cavium_atomic_set(&oct->in_interrupt, 1);

	/* Disable our interrupts for the duration of ISR */
	oct->fn_list.disable_interrupt(oct->chip);

	oct->stats.interrupts++;

	cavium_atomic_inc(&oct->interrupts);

	if(intr64 & CN68XX_INTR_ERR)
		cn68xx_handle_pcie_error_intr(oct, intr64);

	if(intr64 & CN68XX_INTR_PKT_DATA)
		cn68xx_droq_intr_handler(oct);

#ifdef CVMCS_DMA_IC

	if (intr64 & (CN68XX_INTR_DMA0_COUNT | CN68XX_INTR_DMA0_TIME)) {
		uint64_t loc_dma_cnt = octeon_read_csr(oct, CN68XX_DMA_CNT_START);
		cavium_print (PRINT_DEBUG, "DMA Count %lx timer: %lx intr: %lx\n", 
				octeon_read_csr(oct, CN68XX_DMA_CNT_START), 
				octeon_read_csr(oct, CN68XX_DMA_TIM_START), intr64);
		cavium_atomic_add(loc_dma_cnt, &oct->dma_cnt_to_process);
		/* Acknowledge from host, we are going to read 
		   loc_dma_cnt packets from DMA. */
		octeon_write_csr (oct, CN68XX_DMA_CNT_START, loc_dma_cnt);
		cavium_tasklet_schedule(&oct->comp_tasklet);
	}
#else
	if(intr64 & (CN68XX_INTR_DMA0_FORCE|CN68XX_INTR_DMA1_FORCE))
		cavium_tasklet_schedule(&oct->comp_tasklet);
#endif

	if((intr64 & CN68XX_INTR_DMA_DATA) && (oct->dma_ops.intr_handler)) {
		oct->dma_ops.intr_handler((void *)oct, intr64);
	}

	/* Clear the current interrupts */
	OCTEON_WRITE64(cn68xx->intr_sum_reg64, intr64);

	/* Re-enable our interrupts  */
	oct->fn_list.enable_interrupt(oct->chip);

	cavium_atomic_set(&oct->in_interrupt, 0);

	return CVM_INTR_HANDLED;
~~~
可以看出在中断源有很多种，通过判断中断状态寄存器的各个位来区分不同的中断源。
这里主要处理3种中断类型：

- `CN68XX_INTR_ERR`: 错误和异常。
- `CN68XX_INTR_PKT_DATA`: 接收数据包。
- `CN68XX_INTR_DMA0_COUNT`, `CN68XX_INTR_DMA0_COUNT`, `CN68XX_INTR_DMA_DATA`: dma传输

这里我们只关心数据包的接收中断, 现在我们只是知道有数据包收到了，但是不知是哪个queue收到的数据包，
因为所有的queue是公用一个中断号的。
这里的有必要说下，关于中断的2个触发条件，在现在的配置中，我们只配置了时间因素，而没有用数据包的个数这个因素。
所以这里有个trick：我们通过读取时间中断的寄存器，就可以判断有哪些queue收到了数据包。

~~~
octeon_intr_handler
 ->cn68xx_interrupt_handler
  ->cn68xx_droq_intr_handler:

	droq_time_mask = octeon_read_csr(oct, CN68XX_SLI_PKT_TIME_INT);

	droq_mask = droq_time_mask;

	for (oq_no = 0; oq_no < oct->num_oqs; oq_no++) {
		if ( !(droq_mask & (1 << oq_no)) )
			continue;

		droq = oct->droq[oq_no];
		pkt_count = octeon_droq_check_hw_for_pkts(oct, droq);
		if(pkt_count) {
			...
			cavium_wakeup(&droq->wc);
			...
		}
	}
~~~
可以看出，如果某条queue有数据包需要处理，就会wake up该queue上的一个内核线程去处理后续的工作。

这个线程的回调函数如下：

~~~
oct_droq_thread:

	while(!droq->stop_thread && !cavium_kthread_signalled())  {
		
		while(cavium_atomic_read(&droq->pkts_pending))
			octeon_droq_process_packets(droq->oct_dev, droq);

		cavium_sleep_atomic_cond(&droq->wc, &droq->pkts_pending);
	}

oct_droq_thread
 ->octeon_droq_process_packets:

	if(droq->fastpath_on)
		pkts_processed = octeon_droq_fast_process_packets(oct, droq, pkt_count);
	else {
		pkts_processed = octeon_droq_slow_process_packets(oct, droq, pkt_count);
	}
~~~
这里有一个快路和慢路的概念：

- 快路：如果上层注册了一个快路回调函数，便进入快路模式，调用其注册的回调函数。

- 慢路：如果没有快路回调函数注册，则进入慢路模式。
在慢路模式中会根据数据包头部的`opcode`,间接的调用与这对应的回调函数，如果没有，则丢弃该数据包

~~~
oct_droq_thread
 ->octeon_droq_process_packets
  ->octeon_droq_slow_process_packets
   ->octeon_droq_dispatch_pkt:

	disp_fn = octeon_get_dispatch(oct, (uint16_t)resp_hdr->opcode);
	if(disp_fn) {
		rinfo = octeon_create_recv_info(oct, droq, cnt, droq->host_read_index);
		if(rinfo) {
			struct __dispatch *rdisp = rinfo->rsvd;
			rdisp->rinfo    = rinfo;
			rdisp->disp_fn  = disp_fn;
			*((uint64_t *)&rinfo->recv_pkt->resp_hdr) = *((uint64_t *)resp_hdr);
			cavium_list_add_tail(&rdisp->list, &droq->dispatch_list);
		} else {
			droq->stats.dropped_nomem++;
		}
	} else {
		droq->stats.dropped_nodispatch++;
	}  /* else (dispatch_fn ... */
~~~
这里之所以说是间接的调用，是因为所有的回调函数都是之后被每条queue上的内核线程统一调用的：

~~~
oct_droq_thread
 ->octeon_droq_process_packets:

	cavium_list_for_each_safe(tmp, tmp2, &droq->dispatch_list) {
		struct __dispatch *rdisp = (struct __dispatch *)tmp;
		cavium_list_del(tmp);
		rdisp->disp_fn(rdisp->rinfo,
			octeon_get_dispatch_arg(oct, rdisp->rinfo->recv_pkt->resp_hdr.opcode));
	}
~~~

这里由于上层注册了回调函数，进入快路模式，不过在调用注册的回调函数之前，先要做一个基本检查，
在这里只是检查`info entry`中的数据包长度，如果产度为0，则丢弃。

~~~
oct_droq_thread
 ->octeon_droq_process_packets
  ->octeon_droq_fast_process_packets:

	for(pkt = 0; pkt < pkt_count; pkt++)   {

		info = &(droq->info_list[droq->host_read_index]);

		if(!info->length)  {
			cavium_error("OCTEON:DROQ[%d] idx: %d len:0, pkt_cnt: %d \n",
			             droq->q_no, droq->host_read_index, pkt_count);
			cavium_error_print_data((uint8_t *)info, OCT_DROQ_INFO_SIZE);
			pkt++;
			break;
		}
		...
	}
~~~

如果数据包的长度不为0，则调用注册的回调函数

~~~
oct_droq_thread
 ->octeon_droq_process_packets
  ->octeon_droq_fast_process_packets:

			if(droq->ops.fptr)
				droq->ops.fptr(oct->octeon_id, nicbuf, pkt_len, resp_hdr);
			else
				free_recv_buffer(nicbuf);

oct_droq_thread
 ->octeon_droq_process_packets
  ->octeon_droq_fast_process_packets
   ->octnet_push_packet:

	struct sk_buff     *skb   = (struct sk_buff *)skbuff;
	octnet_os_devptr_t *pndev = (octnet_os_devptr_t *)octprops[octeon_id]->pndev[resp_hdr->dest_qport];

	if(pndev) {
		octnet_priv_t  *priv  = GET_NETDEV_PRIV(pndev);
	
		/* Do not proceed if the interface is not in RUNNING state. */
		if( !(cavium_atomic_read(&priv->ifstate) & OCT_NIC_IFSTATE_RUNNING)) {
			free_recv_buffer(skb);
			priv->stats.rx_dropped++;
			return;
		}

		skb->dev       = pndev;
		skb->protocol  = eth_type_trans(skb, skb->dev);
		skb->ip_summed = CHECKSUM_NONE;

		if(netif_rx(skb) != NET_RX_DROP) {
			priv->stats.rx_bytes += len;
			priv->stats.rx_packets++;
			pndev->last_rx  = jiffies;
		} else {
			priv->stats.rx_dropped++;
		}

	} else  {
		free_recv_buffer(skb);
	}
~~~
这里的代码比较简单，只是单纯的将数据包上送至协议栈，不做其他的处理。
下面的工作就交给了内核协议栈去完成，最终，上层应用表现为从虚拟网口收到数据包。

至此，数据的接收流程结束。
