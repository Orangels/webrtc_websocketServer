'use strict'

//本地视频预览窗口
var localVideo = document.querySelector('video#localvideo');

//远端视频预览窗口
var remoteVideo = document.querySelector('video#remotevideo');

//查看Offer文本窗口
var offer = document.querySelector('textarea#offer');

//查看Answer文本窗口
var answer = document.querySelector('textarea#answer');

var pcConfig = {
    'iceServers': [{
        //TURN服务器地址
        'urls': 'turn:43.143.227.135:3478',
        //TURN服务器用户名
        'username': "ghb",
        //TURN服务器密码
        'credential': "moonshine"
    }],
    //默认使用relay方式传输数据
    "iceTransportPolicy": "all",
    "iceCandidatePoolSize": "0"
};

//websocket连接
var conn;

//会话描述
var session;

//本地视频流
var localStream = null;

//远端视频流
var remoteStream = null;

//PeerConnection
var pc = null;

//offer描述
var offerdesc = null;

/**
 * 功能: 判断此浏览器是在PC端,还是移动端。
 * 返回值:  false, 说明当前操作系统是移动端
 *          true, 说明当前的操作系统是PC端。
 */

function IsPC() {
    var userAgentInfo = navigator.userAgent;
    var Agents = ["Android", "iPhone", "SymbianOS", "Windows Phone", "iPad", "iPod"];
    var flag = true;
    for (var v = 0; v < Agents.length; v++) {
        if (userAgentInfo.indexOf(Agents[v]) > 0) {
            flag = false;
            break;
        }
    }
    return flag;
}

/**
 * 功能: 判断是Android端还是iOS端。
 * 返回值: true，说明是Android端
 *        false，说明是iOS端。
 */
function IsAndroid() {
    var u = navigator.userAgent, app = navigator, appVersion;
    var isAndroid = u.indexOf('Android') > -1 || u.indexOf('Linux') > -1;
    var isIOS = !!u.match(/\(i[^;]+;( U;) ? CPU.+Mac OS X/);
    if (isAndroid) {
        //这个是Android系统
        return true;
    }
    if (isIOS) {
        //这个是iOS系统
        return false;
    }
}

/**
 * 功能: 向对端发消息
 * 返回值: 无
 */
function sendMessage(roomid, data) {
    console.log('send message to other end', roomid, data);
    if (!socket) {
        console.log('socket is null');
    }
    socket.emit('message', roomid, data);
}

/**
 *功能: 与信令服务器建立socket.io连接;并根据信令更新状态机。
 *返回值: 无
 */
function conn() {
    //连接信令服务器
    socket = io.connect();
    //joined'消息处理函数
    socket.on('joined', (roomid, id) => {
        console.log('receive joined message!', roomid, id);
        //状态机变更为joined'
        state = 'joined';
        /**
         * 如果是Mesh方案，第一个人不该在这里创建
         * peerConnection，而是要等到所有端都收到一个'otherjoin'消息时再创建
         */
        //创建PeerConnection 并绑定音视频轨
        createPeerConnection();
        bindTracks();
        //设置button状态
        btnConn.disabled = true;
        btnLeave.disabled = false;
        console.log('receive joined message, state = ', state);
    });

    //otherjoin消息处理函数
    socket.on('other_join', (roomid) => {
        console.log('receive joined message:', roomid, state);
        //如果是多人，每加入一个人都要创建一个新的 PeerConnection
        if (state === 'joined_unbind') {
            createPeerConnection();
            bindTracks();
        }
        //状态机变更为 joined_conn
        state = 'joined_conn';
        //开始“呼叫”对方
        call1();
        console.log('receive other_join message, state = ', state);
    });
   
    //full消息处理函数
    socket.on('full', (roomid, id) => {
        console.log('receive full message', roomid, id);
        //关闭gocket.io连接
        socket.disconnect();
        //挂断“呼叫”
        hangup();
        //关闭本地媒体
        closeLocalMedia();
        //状态机变更为 leaved
        state = 'leaved';
        console.log('receive full message, state = ', state);
        alert('the room is full!');
    });

    //leaved消息处理函数
    socket.on('left', (roomid, id) => {
        console.log('receive leaved message', roomid, id);
        //状态机变更为leaved
        state = 'leaved'
        //关闭socket.io连接
        socket.disconnect();
        console.log('receive leaved message, state = ', state);
        //改变button状态
        btnConn.disabled = false;
        btnLeave.disabled = true;
    });

    //bye消息处理函数
    socket.on('bye', (room, id) => {
        console.log('receive bye message', roomid, id);
        /**
         * 当是Mesh方案时，应该带上当前房间的用户数，
         * 如果当前房间用户数不小于 2，则不用修改状态，
         * 并且关闭的应该是对应用户的PeerConnection。
         * 在客户端应该维护一张PeerConnection表，它是
         * key:value的格式，key=userid，value=peerconnection
         */
        //状态机变更为 joined_unbind
        state = 'joined_unbind';
        //挂断“呼叫”
        hangup();
        offer.value = '';
        answer.value = '';
        console.log('receive bye message, state=', state);
    });

    //socket.io连接断开处理函数
    socket.on('disconnect', (socket) => {
        console.log('receive disconnect message!', roomid);
        if (!(state === 'leaved')) {
            //挂断“呼叫”
            hangup();
            //关闭本地媒体
            closeLocalMedia();
        }
        //状态机变更为 leaved
        state = 'leaved';
    });

    //收到对端消息处理函数
    socket.on('message', (roomid, data) => {
        console.log('receive message!', roomid, data);
        if( data === null || data === undefined) {
            console.error('the message is invalid!');
            return;
        }
        //如果收到的SDP是offer
        if (data.hasOwnProperty('type') && data.type === 'offer') {
            offer.value = data.sdp;
            //进行媒体协商
            pc.setRemoteDescription(new RTCSessionDescription(data));
            //创建answer
            pc.createAnswer()
                .then(getAnswer)
                .catch(handleAnswerError);
            //如果收到的SDP是answer
        } else if (data.hasOwnProperty('type') && data.type == 'answer') {
            answer.value = data.sdp;
            //进行媒体协商
            pc.setRemoteDescription(new RTCSessionDescription(data));
            //如果收到的是Candidate消息
        } else if (data.hasOwnProperty('type') && data.type === 'candidate') {
            var candidate = new RTCIceCandidate({
                sdpMLineIndex: data.label,
                candidate: data.candidate
            });
            //将远端Candidate消息添加到PeerConnection中
            pc.addIceCandidate(candidate);
        } else {
            console.log('the message is invalid!', data);
        }
    });

    //从url中获取roomid
    roomid = getQueryVariable('room');

    //发送'join'消息
    socket.emit('join', roomid);

    return true;
}

/**
 * 功能: 打开音视频设备成功时的回调函数
 * 返回值: 永远为true
 */
function getMediaStream(stream) {
    //将从设备上获取到的音视频track添加到localStream中
    if (localStream) {
        stream.getAudioTracks().forEach((track) => {
            localStream.addTrack(track);
            stream.removeTrack(track);
        });
    } else {
        localStream = stream;
    }
    //本地视频标签与本地流绑定
    localVideo.srcObject = localStream;

    //创建PeerConnection 并绑定音视频轨
    createPeerConnection();
    bindTracks();
}

/**
 * 功能: 错误处理函数
 *
 * 返回值: 无
 */
function handleError(err) {
    console.error('Failed to get Media Stream!', err);
}

/**
 *功能: 打开音视频设备
 *返回值: 无
 */
function startLocalMedia() {
    if (!navigator.mediaDevices ||
        !navigator.mediaDevices.getUserMedia) {
        console.error('the getUserMedia is not supported!');
        return;
    } else {
        var constraints;
        constraints = {
            video: true,
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                autoGainControl: true
            }
        };

        navigator.mediaDevices.getUserMedia(constraints)
            .then(getMediaStream)
            .catch(handleError);
    }
}

/**
 * 功能: 获得远端媒体流
 * 返回值: 无
 */
function getRemoteStream(e) {
    //存放远端视频流
    remoteStream = e.streams[0];
    //远端视频标签与远端视频流绑定
    remoteVideo.srcObject = e.streams[0];
}

/**
 * 功能: 处理Offer错误
 * 返回值: 无
 */
function handleOfferError(err) {
    console.error('Failed to create offer:', err);
}

/**
*功能:处理Answer错误
*返回值:无
*/
function handleAnswerError(err) {
    console.error('Failed to create answer;', err);
}

/**
*功能:获取Answer SDP 描述符的回调函数
*
*返回值:无
*/
function getAnswer(desc) {
    //设置Answer
    pc.setLocalDescription(desc);
    //将Answer显示出来
    answer.value = desc.sdp;
    //将AnswerSDP发送给对端
    let req = new Object;
    req.cmd = 'CMD_WEBRTC';
    req.data = desc;
    let str = JSON.stringify(req);
    console.log(str);
    conn.send(str);
}

/**
*功能:获取Offer SDP 描述符的回调函数
*返回值:无
*/
function getOffer(desc) {
    //设置Offer
    pc.setLocalDescription(desc);
    //将Offer显示出来
    offer.value = desc.sdp;
    offerdesc = desc;

    console.log(offerdesc)

    //将 OfferSDP发送给对端
    let req = new Object;
    req.cmd = 'CMD_WEBRTC';
    req.data = desc;
    let str = JSON.stringify(req);
    console.log(str);
    conn.send(str);
}

/**
* 功能: 创建PeerConnection对象
* 返回值: 无
*/
function createPeerConnection() {
    /*
     * 如果是多人的话，在这里要创建一个新的连接
     * 新创建好的要放到一个映射表中
     */
    //key=userid, value=peerconnection
    console.log('create RTCPeerConnection!');
    if (!pc) {
        //创建PeerConnection对象
        pc = new RTCPeerConnection(pcConfig);
        //当收集到Candidate后
        pc.onicecandidate = (event) => {
            if (event.candidate) {
                console.log("candidate" + JSON.stringify(event.candidate.toJSON()));
                //将Candidate发送给对端
                let req = new Object;
                req.cmd = 'CMD_WEBRTC';
                req.data = {
                    type: 'candidate',
                    label: event.candidate.sdpMLineIndex,
                    id: event.candidate.sdpMid,
                    candidate: event.candidate.candidate
                };
                let str = JSON.stringify(req);
                console.log(str);
                conn.send(str);
            } else {
                console.log('this is the end candidate');
            }
        }
        /**
         * 当PeerConnection对象收到远端音视频流时
         * 触发ontrack事件，并回调getRemoteStream函数
         */
        pc.ontrack = getRemoteStream;
    } else {
        console.log('the pc have be created!');
    }
    return;
}

/**
* 功能: 将音视频track绑定到PeerConnection对象中
* 返回值: 无
*/
function bindTracks() {
    console.log('bind tracks into RTCPeerConnection!');
    if (pc === null && localStream === undefined) {
        console.error('pc is null or undefined!');
        return;
    }

    if (localStream === null && localStream === undefined) {
        console.error('localstream is null or undefined!');
        return;
    }

    //将本地音视频流中所有的track添加到PeerConnection对象中
    localStream.getTracks().forEach((track)=>{
        pc.addTrack(track, localStream);
    });
}

/**
 * 功能: 开启“呼叫”
 * 返回值: 无
 */
function videoCall() {
    var offerOptions = {
        offerToReceiveAudio: 1,
        offerToReceiveVideo: 1
    };
    /**
     * 创建Offer，
     * 如果成功，则回调getoffer()方法
     * 如果失败，则回调handleofferError()方法
     */
    pc.createOffer(offerOptions)
        .then(getOffer)
        .catch(handleOfferError);
}

/**
 * 功能: 挂断“呼叫”
 * 返回值: 无
 */
function hangup() {
    if (!pc) {
        return;
    }

    //发送“挂断”消息
    let r = new Object;
    r.cmd = "CMD_HANGUP";
    let str = JSON.stringify(r);
    conn.send(str);

    /*
    offerdesc = null;

    //将PeerConnection连接关掉
    pc.close();
    pc = null;
    */
}

/**
 * 功能:关闭本地媒体
 * 返回值:无
 */
function closeLocalMedia() {
    if (!(localStream === null || localStream === undefined)) {
        //遍历每个track，并将其关闭
        localStream.getTracks().forEach((track) => {
            track.stop();
        });
    }
    localStream = null;
}

/**
*功能:离开房间
*返回值:无
*/
function leave() {
    //向信令服务器发送leave消息
    socket.emit('leave', roomid);
    //挂断“呼叫”
    hangup();
    //关闭本地媒体
    closeLocalMedia();
    offer.value = '';
    answer.value = '';
}

