(function () {
  "use strict";

  var KEY = "stale-lang";
  var LANGS = ["en", "ar", "ru", "zh"];
  var BASE = "https://stale.elghaly.dev/";

  // English HTML fragments (static only — never read from the DOM into innerHTML).
  var EN_HTML = {
    lim2:
      "Measures slot lag with <code>getSlot</code> at <code>processed</code> commitment. It does not measure the age of account data, quotes, or any other answer.",
    lim3:
      "Lag in milliseconds is slots × 400&nbsp;ms, an estimate from the nominal slot time, not a measured duration.",
    lim5:
      "The default reference, <code>api.mainnet-beta.solana.com</code>, is rate-limited. A refused or slow call never produces FRESH. It can still end in STALE when at least two earlier samples, answered by both endpoints, already proved the lag; the verdict line then says <code>degraded=true</code> and prints <code>lag=unknown</code> if the last sample went unanswered.",
    lim7:
      "Read-only proxy — <code>sendTransaction</code> and <code>requestAirdrop</code> are refused in any letter case, and a request that names its method twice or in two spellings is refused before it is forwarded.",
    lim8:
      "The proxy's <code>X-Stale-*</code> headers describe the latest background sample, not the response they arrive on."
  };

  var T = {
    ar: {
      __title: "stale — تأخر فتحات RPC سولانا، مُقاس",
      __meta: "stale — كم فتحة يتأخر RPC سولانا عن نقطة مرجعية. فحص تأخر الفتحات للقراءة فقط ووكيل. MIT.",
      skip: "تخطّ إلى المحتوى",
      hero_tag: "كم فتحة يتأخر RPC سولانا لديك عن نقطة مرجعية؟ stale يقيس ذلك مرة كل ثانية — ويقول UNKNOWN عندما لا يستطيع الحكم.",
      install_h: "تثبيت",
      example_badge: "اصطناعي",
      example_h: "مثال على المخرجات",
      why_badge: "من سجلات بوتنا · لم يقِسها stale",
      why_h: "لماذا بنيناه",
      why_p1: "بوت التداول لدينا وسم كل عرض hop-1 قيّمه خلال نصف ساعة بأنه «fresh» — 283 من 283. الأعمار التي سجّلها بجانب هذا الوصف من أربع إلى ثماني ثوانٍ. التسمية من كودنا، لا من أي مزوّد، وكانت خاطئة: البوت اعتبر العرض fresh لأنه مخزّن مؤقتاً دون التحقق من عمره. اكتشفنا ذلك بقراءة سجلات البوت والحكم على الكلمة بالرقم بجانبها. stale هو ذلك الفحص، في أداة يمكنك توجيهها إلى أي RPC.",
      why_p2: "stale لم ينتج الأرقام أدناه — إنها أعمار عروض كتبها بوتنا إلى سجله، بالمللي ثانية، خلال نصف ساعة من المسح. تشغيل stale يبلّغ عن شيء آخر: التأخر بالفتحات مقابل نقطة مرجعية.",
      stat1: "عروض وسمها بوتنا fresh",
      stat2: "وسيط العمر المسجّل",
      stat3: "أقصى عمر مسجّل",
      stat4: "الحد الذي وضعناه",
      limits_h: "حدود",
      lim1: "يقارن بنقطة مرجعية واحدة — إذا تأخر الطرفان بالقدر نفسه، لا يستطيع stale التمييز.",
      lim2: "يقيس تأخر الفتحات عبر <code>getSlot</code> عند التزام <code>processed</code>. لا يقيس عمر بيانات الحساب أو العروض أو أي إجابة أخرى.",
      lim3: "التأخر بالمللي ثانية = فتحات × 400&nbsp;ms، تقدير من زمن الفتحة الاسمي وليس مدة مقاسة.",
      lim4: "الحكم يقيّم التأخر في آخر عينة من النافذة. هدف كان متأخراً ثم لحق في النهاية يقرأ FRESH.",
      lim5: "المرجع الافتراضي <code>api.mainnet-beta.solana.com</code> محدود المعدّل. استدعاء مرفوض أو بطيء لا ينتج FRESH أبداً. قد ينتهي STALE عندما أثبتت عينتان سابقتان على الأقل، أجاب عليهما الطرفان، التأخر؛ يقول سطر الحكم <code>degraded=true</code> ويطبع <code>lag=unknown</code> إذا لم تُجَب العينة الأخيرة.",
      lim6: "نقطة نهاية مقيسة ضد نفسها تُرفض — التأخر ضد نفسك دائماً صفر.",
      lim7: "وكيل للقراءة فقط — <code>sendTransaction</code> و <code>requestAirdrop</code> مرفوضان بأي حالة أحرف، وطلب يسمّي method مرتين أو بإملائين يُرفض قبل إعادة التوجيه.",
      lim8: "رؤوس <code>X-Stale-*</code> للوكيل تصف آخر عينة خلفية، وليس الاستجابة التي تصل عليها.",
      lim9: "بث WebSocket ينتظر v0.2.",
      footer_keys: "لا مفاتيح، لا محفظة، للقراءة فقط",
      footer_contact: "Email first: support@elghaly.dev. Calls only by appointment.",
      notfound_h: "لا توجد صفحة كهذه.",
      notfound_home: "العودة إلى stale",
      notfound_code: "404"
    },
    ru: {
      __title: "stale — отставание слотов RPC Solana, измерено",
      __meta: "stale — на сколько слотов RPC Solana отстаёт от эталонной точки. Только чтение: проверка отставания слотов и прокси. MIT.",
      skip: "Перейти к содержанию",
      hero_tag: "На сколько слотов ваш RPC Solana отстаёт от эталона? stale меряет раз в секунду — и говорит UNKNOWN, когда не может судить.",
      install_h: "Установка",
      example_badge: "Синтетика",
      example_h: "Пример вывода",
      why_badge: "Из логов нашего бота · stale не измерял",
      why_h: "Зачем мы это сделали",
      why_p1: "Наш торговый бот пометил каждую hop-1 котировку за полчаса как «fresh» — 283 из 283. Возрасты в логе рядом с этим словом — от четырёх до восьми секунд. Метку ставил наш код, не провайдер, и она была неверной: бот считал котировку fresh, потому что она была в кэше, без проверки возраста. Мы увидели это в логах и сверили слово с числом рядом. stale — эта проверка, оформленная как инструмент для любого RPC.",
      why_p2: "stale не выдал цифры ниже — это возрасты котировок из лога бота в миллисекундах за полчаса сканов. Запуск stale сообщает другое: отставание в слотах относительно эталона.",
      stat1: "Котировок бот пометил fresh",
      stat2: "Медианный залогированный возраст",
      stat3: "Макс. залогированный возраст",
      stat4: "Наш лимит",
      limits_h: "Ограничения",
      lim1: "Сравнение с одним эталоном — если оба отстают одинаково, stale не различит.",
      lim2: "Меряет отставание слотов через <code>getSlot</code> с commitment <code>processed</code>. Не меряет возраст данных аккаунта, котировок или других ответов.",
      lim3: "Отставание в мс = слоты × 400&nbsp;ms — оценка по номинальному времени слота, не измеренная длительность.",
      lim4: "Вердикт по отставанию на последней выборке окна. Цель отставала и догнала к концу — FRESH.",
      lim5: "Эталон по умолчанию <code>api.mainnet-beta.solana.com</code> с лимитом. Отказ или медленный вызов никогда не даёт FRESH. Может закончиться STALE, если минимум две более ранние выборки с ответами обоих концов уже доказали отставание; строка вердикта тогда <code>degraded=true</code> и <code>lag=unknown</code>, если последняя выборка без ответа.",
      lim6: "Эндпоинт против самого себя отклоняется — отставание от себя всегда ноль.",
      lim7: "Прокси только для чтения — <code>sendTransaction</code> и <code>requestAirdrop</code> отклоняются в любом регистре; запрос с method дважды или в двух написаниях отклоняется до проксирования.",
      lim8: "Заголовки <code>X-Stale-*</code> прокси описывают последнюю фоновую выборку, а не ответ, на котором они пришли.",
      lim9: "WebSocket fan-out ждёт v0.2.",
      footer_keys: "без ключей, без кошелька, только чтение",
      footer_contact: "Email first: support@elghaly.dev. Calls only by appointment.",
      notfound_h: "Такой страницы нет.",
      notfound_home: "Назад к stale",
      notfound_code: "404"
    },
    zh: {
      __title: "stale — 测量 Solana RPC 槽位滞后",
      __meta: "stale — Solana RPC 相对参考端点落后多少槽位。只读槽位滞后检查与代理。MIT。",
      skip: "跳到正文",
      hero_tag: "你的 Solana RPC 比参考端点落后多少槽？stale 每秒测量一次 — 无法判断时返回 UNKNOWN。",
      install_h: "安装",
      example_badge: "合成示例",
      example_h: "输出示例",
      why_badge: "来自我们 bot 的日志 · 非 stale 测量",
      why_h: "为什么做它",
      why_p1: "我们的交易 bot 在半小时内把定价的 hop-1 报价全部标为「fresh」— 283/283。日志里该词旁边的年龄从四到八秒。标签来自我们的代码，不是任何提供商，而且错了：bot 因缓存而称报价 fresh，未检查年龄。我们读 bot 日志，用旁边的数字检验这个词。stale 就是把这类检查做成可指向任意 RPC 的工具。",
      why_p2: "stale 未产生下方数字 — 它们是 bot 在半小时扫描中写入日志的报价年龄（毫秒）。stale 运行报告的是另一件事：相对参考端点的槽位滞后。",
      stat1: "bot 标为 fresh 的报价",
      stat2: "记录年龄中位数",
      stat3: "记录的最大年龄",
      stat4: "我们设定的上限",
      limits_h: "限制",
      lim1: "只与一个参考比较 — 若两端点同等落后，stale 无法区分。",
      lim2: "用 <code>getSlot</code>、<code>processed</code> 承诺测量槽位滞后。不测量账户数据、报价或其他响应的年龄。",
      lim3: "毫秒滞后 = 槽数 × 400&nbsp;ms，来自名义槽时间的估计，非实测时长。",
      lim4: "裁决依据窗口内最后一次采样。目标曾落后但在结束时追上则读为 FRESH。",
      lim5: "默认参考 <code>api.mainnet-beta.solana.com</code> 有速率限制。拒绝或缓慢调用永不产生 FRESH。若至少两次更早采样（两端均有应答）已证明滞后，仍可能 STALE；裁决行 <code>degraded=true</code>，末次采样无应答则 <code>lag=unknown</code>。",
      lim6: "端点不能测量自身 — 相对自己滞后恒为零。",
      lim7: "只读代理 — <code>sendTransaction</code> 与 <code>requestAirdrop</code> 任意大小写均拒绝；method 重复或两种拼写的请求在转发前拒绝。",
      lim8: "代理的 <code>X-Stale-*</code> 头描述最新后台采样，而非它们所附的响应。",
      lim9: "WebSocket 扇出等待 v0.2。",
      footer_keys: "无密钥、无钱包、只读",
      footer_contact: "Email first: support@elghaly.dev. Calls only by appointment.",
      notfound_h: "没有此页面。",
      notfound_home: "返回 stale",
      notfound_code: "404"
    }
  };

  var nodes = document.querySelectorAll("[data-i18n]");
  var htmlNodes = document.querySelectorAll("[data-i18n-html]");
  var en = { __title: document.title };
  var metaDesc = document.querySelector('meta[name="description"]');
  if (metaDesc) en.__meta = metaDesc.getAttribute("content") || "";
  var i;
  for (i = 0; i < nodes.length; i++) {
    en[nodes[i].getAttribute("data-i18n")] = nodes[i].textContent;
  }

  var buttons = document.querySelectorAll("#langs [data-lang]");
  var root = document.documentElement;

  function htmlString(lang, key) {
    if (lang === "en") return EN_HTML[key];
    var pack = T[lang];
    return pack && pack[key];
  }

  function set(lang, save) {
    if (LANGS.indexOf(lang) < 0) lang = "en";
    var pack = lang === "en" ? en : T[lang];
    for (i = 0; i < nodes.length; i++) {
      var k = nodes[i].getAttribute("data-i18n");
      if (pack[k] != null) nodes[i].textContent = pack[k];
      else if (en[k] != null) nodes[i].textContent = en[k];
    }
    for (i = 0; i < htmlNodes.length; i++) {
      var hk = htmlNodes[i].getAttribute("data-i18n-html");
      var html = htmlString(lang, hk);
      if (html != null) htmlNodes[i].innerHTML = html;
    }
    document.title = pack.__title || en.__title;
    if (metaDesc && (pack.__meta || en.__meta)) {
      metaDesc.setAttribute("content", pack.__meta || en.__meta);
    }
    root.lang = lang === "zh" ? "zh-Hans" : lang;
    root.dir = lang === "ar" ? "rtl" : "ltr";
    for (i = 0; i < buttons.length; i++) {
      buttons[i].setAttribute(
        "aria-pressed",
        buttons[i].getAttribute("data-lang") === lang ? "true" : "false"
      );
    }
    if (save) {
      try {
        localStorage.setItem(KEY, lang);
      } catch (e) { /* ignore */ }
      try {
        var u = new URL(location.href);
        if (lang === "en") u.searchParams.delete("lang");
        else u.searchParams.set("lang", lang);
        history.replaceState(null, "", u.pathname + u.search + u.hash);
      } catch (e2) { /* ignore */ }
    }
  }

  function first() {
    var m = /[?&]lang=(en|ar|ru|zh)/.exec(location.search);
    if (m && LANGS.indexOf(m[1]) >= 0) return m[1];
    try {
      var saved = localStorage.getItem(KEY);
      if (saved && LANGS.indexOf(saved) >= 0) return saved;
    } catch (e) { /* ignore */ }
    var prefs = navigator.languages || [navigator.language || "en"];
    for (i = 0; i < prefs.length; i++) {
      var two = String(prefs[i]).slice(0, 2).toLowerCase();
      if (LANGS.indexOf(two) >= 0) return two;
    }
    return "en";
  }

  for (i = 0; i < buttons.length; i++) {
    buttons[i].addEventListener("click", function () {
      set(this.getAttribute("data-lang"), true);
    });
  }

  set(first(), false);

  window.StaleI18n = {
    langs: LANGS,
    base: BASE,
    setLang: function (lang) { set(lang, true); }
  };
})();
