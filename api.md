# Project

```bash
curl -X 'POST' \
  'https://quran.namaz.live/recordings?learner_id=some_random_993&ayah_id=001001' \
  -H 'accept: application/json' \
  -H 'x-api-key: XXX' \
  -H 'Content-Type: multipart/form-data' \
  -F 'file=@001001_hussary.wav;type=audio/wav'
```

```json
{
  "recording_id": "3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50",
  "status": "queued",
  "task_id": "40631e7e-0de8-4483-a204-009c53729866"
}
```

```bash
curl -X 'GET' \
  'https://quran.namaz.live/recordings?learner_id=some_random_993&recording_ids=3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50' \
  -H 'accept: application/json' \
  -H 'x-api-key: XXX'
```

```json
{
  "recordings": [
    {
      "status": "done",
      "task_id": "40631e7e-0de8-4483-a204-009c53729866",
      "audio_path": "/mnt/shared/3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50.wav",
      "wav_path": "/mnt/shared/3234e1e9-9596-4359-a1f0-a336661aed5d.wav",
      "stage": "completed",
      "recording_id": "3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50",
      "tmp_path": "/mnt/shared/3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50.wav",
      "createdAt": "2025-12-13T16:35:17.279000Z",
      "learner_id": "some_random_993",
      "ayah_id": "001001",
      "text_len": 37,
      "updatedAt": "2025-12-13T16:35:37.543000Z",
      "result": {
        "wer": 0,
        "ops": [
          {
            "hyp_clean": "بسم",
            "t_end": 0.522,
            "t_start": 0,
            "ref_ar": "بِسْمِ",
            "ref_clean": "بسم",
            "hyp_ar": "بِسْمِ",
            "op": "C"
          },
          {
            "hyp_clean": "الله",
            "t_end": 1.687,
            "t_start": 1.064,
            "ref_ar": "اللَّهِ",
            "ref_clean": "الله",
            "hyp_ar": "اللَّهِ",
            "op": "C"
          },
          {
            "hyp_clean": "الرحمن",
            "t_end": 3.072,
            "t_start": 2.269,
            "ref_ar": "الرَّحْمَنِ",
            "ref_clean": "الرحمن",
            "hyp_ar": "الرَّحْمَنِ",
            "op": "C"
          },
          {
            "hyp_clean": "الرحيم",
            "t_end": 4.216,
            "t_start": 3.574,
            "ref_ar": "الرَّحِيمِ",
            "ref_clean": "الرحيم",
            "hyp_ar": "الرَّحِيمِ",
            "op": "C"
          }
        ],
        "hypothesis": "بِسْمِ اللَّهِ الرَّحْمَنِ الرَّحِيمِ"
      },
      "duration_sec": 20.084,
      "since_start_sec": 19.709
    }
  ],
  "not_found": []
}
```

```bash
curl -X 'GET' \
  'https://quran.namaz.live/recordings/some_random_993?limit=2&order=desc&order_by=createdAt' \
  -H 'accept: application/json' \
  -H 'x-api-key: XXX'
```

```json
{
  "items": [
    {
      "recording_id": "3cbb065e-4fb2-41f2-ad9f-2eb3d2366a50",
      "learner_id": "some_random_993",
      "status": "done",
      "createdAt": "2025-12-13T16:35:17.279000Z",
      "updatedAt": "2025-12-13T16:35:37.543000Z",
      "result": {
        "wer": 0,
        "ops": [
          {
            "hyp_clean": "بسم",
            "t_end": 0.522,
            "t_start": 0,
            "ref_ar": "بِسْمِ",
            "ref_clean": "بسم",
            "hyp_ar": "بِسْمِ",
            "op": "C"
          },
          {
            "hyp_clean": "الله",
            "t_end": 1.687,
            "t_start": 1.064,
            "ref_ar": "اللَّهِ",
            "ref_clean": "الله",
            "hyp_ar": "اللَّهِ",
            "op": "C"
          },
          {
            "hyp_clean": "الرحمن",
            "t_end": 3.072,
            "t_start": 2.269,
            "ref_ar": "الرَّحْمَنِ",
            "ref_clean": "الرحمن",
            "hyp_ar": "الرَّحْمَنِ",
            "op": "C"
          },
          {
            "hyp_clean": "الرحيم",
            "t_end": 4.216,
            "t_start": 3.574,
            "ref_ar": "الرَّحِيمِ",
            "ref_clean": "الرحيم",
            "hyp_ar": "الرَّحِيمِ",
            "op": "C"
          }
        ],
        "hypothesis": "بِسْمِ اللَّهِ الرَّحْمَنِ الرَّحِيمِ"
      },
      "ayah_id": "001001"
    }
  ],
  "next_page_token": null
}
```
